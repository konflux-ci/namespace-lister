package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceTypeLabelKey       string = "konflux-ci.dev/type"
	NamespaceTypeUserLabelValue string = "user"
)

var _ = Describe("Authorizing requests", Serial, Ordered, func() {
	username := "user"
	var restConfig *rest.Config
	var c client.Client
	var ans []client.Object
	var cacheCfg *cacheConfig

	BeforeAll(func(ctx context.Context) {
		var err error

		// prepare scheme
		s := runtime.NewScheme()
		utilruntime.Must(corev1.AddToScheme(s))
		utilruntime.Must(rbacv1.AddToScheme(s))

		// get kubernetes client config
		restConfig = ctrl.GetConfigOrDie()
		restConfig.QPS = 500
		restConfig.Burst = 500

		// build kubernetes client
		c, err = client.New(restConfig, client.Options{Scheme: s})
		utilruntime.Must(err)

		// create resources
		err, ans, _ = createResources(ctx, c, username, 300, 800, 1200)
		utilruntime.Must(err)

		log.Println("allowed namespaces", len(ans))
	})

	BeforeEach(func(ctx context.Context) {
		// create cache
		ls, err := labels.Parse(fmt.Sprintf("%s=%s", NamespaceTypeLabelKey, NamespaceTypeUserLabelValue))
		utilruntime.Must(err)
		cacheCfg = &cacheConfig{restConfig: restConfig, namespacesLabelSector: ls}
	})

	It("efficiently authorize on a huge environment with cached accesses", Serial, Label("perf"), func(ctx context.Context) {
		// new gomega experiment
		experiment := gmeasure.NewExperiment("Authorizing Request")

		// Register the experiment as a ReportEntry - this will cause Ginkgo's reporter infrastructure
		// to print out the experiment's report and to include the experiment in any generated reports
		AddReportEntry(experiment.Name, experiment)

		// create cache, namespacelister, and handler
		cache, err := BuildAndStartResourceCache(ctx, cacheCfg)
		utilruntime.Must(err)
		c, err := buildAndStartSynchronizedAccessCache(ctx, cache, nil)
		utilruntime.Must(err)

		nl := NewSubjectNamespaceLister(c)
		lnh := NewListNamespacesHandler(nl)

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			var err error
			var nn *corev1.NamespaceList

			// measure ListNamespaces
			experiment.MeasureDuration("internal listing", func() {
				nn, err = nl.ListNamespaces(ctx, username, nil)
			})

			// check results
			if err != nil {
				panic(err)
			}
			if lnn := len(nn.Items); lnn != len(ans) {
				panic(fmt.Errorf("expecting %d namespaces, received %d", len(ans), lnn))
			}
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			rctx := context.WithValue(context.Background(), ContextKeyUserDetails, &authenticator.Response{
				User: &user.DefaultInfo{
					Name: username,
				},
			})
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(rctx)
			w := httptest.NewRecorder()

			// measure http Handler
			experiment.MeasureDuration("http listing", func() {
				lnh.ServeHTTP(w, r)
			})
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.
	})

	It("efficiently synchronizes access cache", Serial, Label("perf"), func(ctx context.Context) {
		// new gomega experiment
		experiment := gmeasure.NewExperiment("Access Cache Synch")

		// Register the experiment as a ReportEntry - this will cause Ginkgo's reporter infrastructure
		// to print out the experiment's report and to include the experiment in any generated reports
		AddReportEntry(experiment.Name, experiment)

		// create resourceCache, namespacelister, and handler
		resourceCache, err := BuildAndStartResourceCache(ctx, cacheCfg)
		utilruntime.Must(err)
		registry := prometheus.NewRegistry()
		c, err := buildAndStartSynchronizedAccessCache(ctx, resourceCache, registry)
		utilruntime.Must(err)

		// check cache is correctly populated with
		// the expected number of subjects and namespaces
		samples, err := expfmt.ExtractSamples(
			&expfmt.DecodeOptions{Timestamp: model.Now()},
			&dto.MetricFamily{Name: ptr("subjects")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(samples).To(HaveLen(5502))

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			var err error

			// measure ListNamespaces
			experiment.MeasureDuration("cache synch", func() {
				err = c.Synch(ctx)
			})

			// check results
			if err != nil {
				panic(err)
			}
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we get the median synch duration from the experiment we just ran
		httpListingStats := experiment.GetStats("cache synch")
		medianDuration := httpListingStats.DurationFor(gmeasure.StatMedian)

		// and assert that it is below a threshold
		Expect(medianDuration).To(BeNumerically("<=", 200*time.Millisecond))
	})
})

func createResources(ctx context.Context, cli client.Client, user string, numAllowedNamespaces, numUnallowedNamespaces, numNonMatchingClusterRoles int) (error, []client.Object, []client.Object) {
	// cluster scoped resources
	mcr, nmcr := matchingClusterRoles(1), nonMatchingClusterRoles(numNonMatchingClusterRoles)
	ans, uns := namespaces("allowed-tenant-", numAllowedNamespaces), namespaces("unallowed-tenant-", numUnallowedNamespaces)
	if err := create(ctx, cli, slices.Concat(mcr, nmcr, ans, uns)); err != nil {
		return fmt.Errorf("could not create cluster scoped resources: %w", err), nil, nil
	}

	// namespace scoped resources
	atr := allowedTenants(user, ans, 10, "ClusterRole", mcr[0].GetName(), "ClusterRole", nmcr[0].GetName())
	utr := unallowedTenants(user, uns, 10, "ClusterRole", mcr[0].GetName(), "ClusterRole", nmcr[0].GetName())
	if err := create(ctx, cli, slices.Concat(atr, utr)); err != nil {
		return fmt.Errorf("could not create namespaced scoped resources: %w", err), nil, nil
	}
	return nil, ans, uns
}

func create(ctx context.Context, cli client.Client, rr []client.Object) error {
	for _, r := range rr {
		if err := cli.Create(ctx, r); client.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}
	return nil
}

func matchingClusterRoles(quantity int) []client.Object {
	crr := make([]client.Object, quantity)
	for i := range quantity {
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("perf-cluster-role-matching-%d", i),
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{corev1.GroupName},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				},
			},
		}
		crr[i] = cr
	}
	return crr
}

func nonMatchingClusterRoles(quantity int) []client.Object {
	crr := make([]client.Object, quantity)
	for i := range quantity {
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "perf-cluster-role-non-matching-",
			},
		}
		crr[i] = cr
	}
	return crr
}

func namespaces(generateName string, quantity int) []client.Object {
	rr := make([]client.Object, quantity)
	for i := range quantity {
		rr[i] = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
				Labels: map[string]string{
					NamespaceTypeLabelKey: NamespaceTypeUserLabelValue,
				},
			},
		}
	}
	return rr
}

func allowedTenants(user string, namespaces []client.Object, pollutingRoleBindings int, matchingRoleRefKind, matchingRoleRefName, nonMatchingRoleRefKind, nonMatchingRoleRefName string) []client.Object {
	rr := make([]client.Object, 0, len(namespaces)*(pollutingRoleBindings+1))
	for _, n := range namespaces {
		// add access role binding
		rr = append(rr, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "allowed-tenant-",
				Namespace:    n.GetName(),
			},
			Subjects: append(randNotPerfTestUsers(5),
				rbacv1.Subject{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}),
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     matchingRoleRefKind,
				Name:     matchingRoleRefName,
			},
		})

		// add pollution
		for range pollutingRoleBindings {
			subjects := slices.Concat(
				[]rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}},
				randNotPerfTestUsers(5),
				randNotPerfTestServiceAccounts(5),
			)

			rr = append(rr, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pollution-",
					Namespace:    n.GetName(),
				},
				Subjects: subjects,
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     nonMatchingRoleRefKind,
					Name:     nonMatchingRoleRefName,
				},
			})
		}
	}
	return rr
}

func unallowedTenants(user string, namespaces []client.Object, pollutingRoleBindings int, matchingRoleRefKind, matchingRoleRefName, nonMatchingRoleRefKind, nonMatchingRoleRefName string) []client.Object {
	rr := make([]client.Object, 0, len(namespaces)*(pollutingRoleBindings+1))
	for _, n := range namespaces {
		// add access role binding
		rr = append(rr, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "non-allowed-tenant-",
				Namespace:    n.GetName(),
			},
			Subjects: randNotPerfTestUsers(5),
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     matchingRoleRefKind,
				Name:     matchingRoleRefName,
			},
		})

		// add pollution
		for range pollutingRoleBindings {
			subjects := slices.Concat(
				[]rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}},
				randNotPerfTestUsers(5),
				randNotPerfTestServiceAccounts(5),
			)

			rr = append(rr, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pollution-",
					Namespace:    n.GetName(),
				},
				Subjects: subjects,
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     nonMatchingRoleRefKind,
					Name:     nonMatchingRoleRefName,
				},
			})
		}
	}
	return rr
}

func randNotPerfTestServiceAccounts(size int) []rbacv1.Subject {
	return randNotPerfTestSubject(size, "", "ServiceAccount")
}

func randNotPerfTestUsers(size int) []rbacv1.Subject {
	return randNotPerfTestSubject(size, rbacv1.GroupName, "User")
}

func randNotPerfTestSubject(size int, apiGroup, kind string) []rbacv1.Subject {
	ss := make([]rbacv1.Subject, size, size)
	for i := range size {
		ss[i] = rbacv1.Subject{
			APIGroup: apiGroup,
			Kind:     kind,
			Name:     fmt.Sprintf("not-the-perf-test-%s-%d", strings.ToLower(kind), rand.Int64()), //nolint:gosec,G404
		}
	}
	return ss
}
