package datadog

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metricsapi "github.com/keptn/lifecycle-toolkit/metrics-operator/api/v1alpha2"
	"github.com/keptn/lifecycle-toolkit/metrics-operator/controllers/common/fake"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const ddPayload = "{\"from_date\":1677736306000,\"group_by\":[],\"message\":\"\",\"query\":\"system.cpu.idle{*}\",\"res_type\":\"time_series\",\"series\":[{\"aggr\":null,\"display_name\":\"system.cpu.idle\",\"end\":1677821999000,\"expression\":\"system.cpu.idle{*}\",\"interval\":300,\"length\":7,\"metric\":\"system.cpu.idle\",\"pointlist\":[[1677781200000,92.37997436523438],[1677781500000,91.46615447998047],[1677781800000,92.05865631103515],[1677782100000,97.49858474731445],[1677782400000,95.95263163248698],[1677821400000,69.67094268798829],[1677821700000,84.78184509277344]],\"query_index\":0,\"scope\":\"*\",\"start\":1677781200000,\"tag_set\":[],\"unit\":[{\"family\":\"percentage\",\"name\":\"percent\",\"plural\":\"percent\",\"scale_factor\":1,\"short_name\":\"%\"},{}]}],\"status\":\"ok\",\"to_date\":1677822706000}"
const ddEmptyPayload = "{\"from_date\":1677736306000,\"group_by\":[],\"message\":\"\",\"query\":\"system.cpu.idle{*}\",\"res_type\":\"time_series\",\"series\":[],\"status\":\"ok\",\"to_date\":1677822706000}"

func TestEvaluateQuery_HappyPath(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(ddPayload))
		require.Nil(t, err)
	}))
	defer svr.Close()

	secretName := "datadogSecret"
	apiKey, apiKeyValue := "DD_CLIENT_API_KEY", "fake-api-key"
	appKey, appKeyValue := "DD_CLIENT_APP_KEY", "fake-app-key"
	apiToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "",
		},
		Data: map[string][]byte{
			apiKey: []byte(apiKeyValue),
			appKey: []byte(appKeyValue),
		},
	}
	fakeClient := fake.NewClient(apiToken)

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	b := true
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			SecretKeyRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Optional: &b,
			},
			TargetServer: svr.URL,
		},
	}

	r, raw, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.Nil(t, e)
	require.Equal(t, []byte(ddPayload), raw)
	require.Equal(t, fmt.Sprintf("%.3f", 84.782), r)
}
func TestEvaluateQuery_WrongPayloadHandling(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("garbage"))
		require.Nil(t, err)
	}))
	defer svr.Close()

	secretName := "datadogSecret"
	apiKey, apiKeyValue := "DD_CLIENT_API_KEY", "fake-api-key"
	appKey, appKeyValue := "DD_CLIENT_APP_KEY", "fake-app-key"
	apiToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "",
		},
		Data: map[string][]byte{
			apiKey: []byte(apiKeyValue),
			appKey: []byte(appKeyValue),
		},
	}
	fakeClient := fake.NewClient(apiToken)

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	b := true
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			SecretKeyRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Optional: &b,
			},
			TargetServer: svr.URL,
		},
	}

	r, raw, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.Equal(t, "", r)
	require.Equal(t, []byte(nil), raw)
	require.NotNil(t, e)
}
func TestEvaluateQuery_MissingSecret(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(ddPayload))
		require.Nil(t, err)
	}))
	defer svr.Close()

	fakeClient := fake.NewClient()

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			TargetServer: svr.URL,
		},
	}

	_, _, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.NotNil(t, e)
	require.ErrorIs(t, e, ErrSecretKeyRefNotDefined)
}
func TestEvaluateQuery_SecretNotFound(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(ddPayload))
		require.Nil(t, err)
	}))
	defer svr.Close()

	fakeClient := fake.NewClient()
	secretName := "datadogSecret"

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	b := true
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			SecretKeyRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Optional: &b,
			},
			TargetServer: svr.URL,
		},
	}

	_, _, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.NotNil(t, e)
	require.True(t, errors.IsNotFound(e))
}
func TestEvaluateQuery_RefNonExistingKey(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(ddPayload))
		require.Nil(t, err)
	}))
	defer svr.Close()

	secretName := "datadogSecret"
	apiKey, apiKeyValue := "I_AM_NOT_DD_CLIENT_API_KEY", "value"
	apiToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "",
		},
		Data: map[string][]byte{
			apiKey: []byte(apiKeyValue),
		},
	}
	fakeClient := fake.NewClient(apiToken)

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	b := true
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			SecretKeyRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Optional: &b,
			},
			TargetServer: svr.URL,
		},
	}

	_, _, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.NotNil(t, e)
	require.True(t, strings.Contains(e.Error(), "secret does not contain DD_CLIENT_API_KEY or DD_CLIENT_APP_KEY"))
}
func TestEvaluateQuery_EmptyPayload(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(ddEmptyPayload))
		require.Nil(t, err)
	}))
	defer svr.Close()

	secretName := "datadogSecret"
	apiKey, apiKeyValue := "DD_CLIENT_API_KEY", "fake-api-key"
	appKey, appKeyValue := "DD_CLIENT_APP_KEY", "fake-app-key"
	apiToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "",
		},
		Data: map[string][]byte{
			apiKey: []byte(apiKeyValue),
			appKey: []byte(appKeyValue),
		},
	}
	fakeClient := fake.NewClient(apiToken)

	kdd := KeptnDataDogProvider{
		HttpClient: http.Client{},
		Log:        ctrl.Log.WithName("testytest"),
		K8sClient:  fakeClient,
	}
	metric := metricsapi.KeptnMetric{
		Spec: metricsapi.KeptnMetricSpec{
			Query: "system.cpu.idle{*}",
		},
	}
	b := true
	p := metricsapi.KeptnMetricsProvider{
		Spec: metricsapi.KeptnMetricsProviderSpec{
			SecretKeyRef: v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Optional: &b,
			},
			TargetServer: svr.URL,
		},
	}

	r, raw, e := kdd.EvaluateQuery(context.TODO(), metric, p)
	require.Nil(t, raw)
	require.Equal(t, "", r)
	require.True(t, strings.Contains(e.Error(), "no values in query result"))

}
