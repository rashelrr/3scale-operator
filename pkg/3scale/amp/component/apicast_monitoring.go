package component

import (
	"fmt"

	"github.com/3scale/3scale-operator/pkg/assets"
	"github.com/3scale/3scale-operator/pkg/common"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (apicast *Apicast) ApicastProductionMonitoringService() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "apicast-production-metrics",
			Labels: apicast.Options.ProductionMonitoringLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "metrics",
					Protocol:   v1.ProtocolTCP,
					Port:       9421,
					TargetPort: intstr.FromInt(9421),
				},
			},
			Selector: map[string]string{"deploymentConfig": "apicast-production"},
		},
	}
}

func (apicast *Apicast) ApicastStagingMonitoringService() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "apicast-staging-metrics",
			Labels: apicast.Options.StagingMonitoringLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "metrics",
					Protocol:   v1.ProtocolTCP,
					Port:       9421,
					TargetPort: intstr.FromInt(9421),
				},
			},
			Selector: map[string]string{"deploymentConfig": "apicast-staging"},
		},
	}
}

func (apicast *Apicast) ApicastProductionServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "apicast-production",
			Labels: apicast.Options.ProductionMonitoringLabels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port:   "metrics",
				Path:   "/metrics",
				Scheme: "http",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: apicast.Options.ProductionMonitoringLabels,
			},
		},
	}
}

func (apicast *Apicast) ApicastStagingServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "apicast-staging",
			Labels: apicast.Options.StagingMonitoringLabels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port:   "metrics",
				Path:   "/metrics",
				Scheme: "http",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: apicast.Options.StagingMonitoringLabels,
			},
		},
	}
}

func ApicastMainAppGrafanaDashboard(ns string) *grafanav1alpha1.GrafanaDashboard {
	data := &struct {
		Namespace string
	}{
		ns,
	}
	return &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicast-mainapp",
			Labels: map[string]string{
				"monitoring-key": common.MonitoringKey,
			},
		},
		Spec: grafanav1alpha1.GrafanaDashboardSpec{
			Json: assets.TemplateAsset("monitoring/apicast-grafana-dashboard-1.json.tpl", data),
			Name: fmt.Sprintf("%s/apicast-grafana-dashboard-1.json", ns),
		},
	}
}

func ApicastServicesGrafanaDashboard(ns string) *grafanav1alpha1.GrafanaDashboard {
	data := &struct {
		Namespace string
	}{
		ns,
	}
	return &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicast-services",
			Labels: map[string]string{
				"monitoring-key": common.MonitoringKey,
			},
		},
		Spec: grafanav1alpha1.GrafanaDashboardSpec{
			Json: assets.TemplateAsset("monitoring/apicast-grafana-dashboard-2.json.tpl", data),
			Name: fmt.Sprintf("%s/apicast-grafana-dashboard-2.json", ns),
		},
	}
}

func ApicastPrometheusRules(ns string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicast",
			Labels: map[string]string{
				"monitoring-key": common.MonitoringKey,
				"prometheus":     "application-monitoring",
				"role":           "alert-rules",
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: fmt.Sprintf("%s/apicast.rules", ns),
					Rules: []monitoringv1.Rule{
						{
							Alert: "ApicastRequestTime",
							Annotations: map[string]string{
								"summary":     "Request on instance {{ $labels.instance }} is taking more than one second to process the requests",
								"description": "High number of request taking more than a second to be processed",
							},
							Expr: intstr.FromString(fmt.Sprintf(`sum(rate(total_response_time_seconds_bucket{namespace='%s', pod=~'apicast-production.*'}[1m])) - sum(rate(upstream_response_time_seconds_bucket{namespace='%s', pod=~'apicast-production.*'}[1m])) > 1`, ns, ns)),
							For:  "2m",
							Labels: map[string]string{
								"severity": "warning",
							},
						},
						{
							Alert: "APICastHttp4xxErrorRate",
							Annotations: map[string]string{
								"summary":     "APICast high HTTP 4XX error rate (instance {{ $labels.instance }})",
								"description": "The number of request with 4XX is bigger than the 5% of total request.",
							},
							Expr: intstr.FromString(fmt.Sprintf(`sum(rate(apicast_status{namespace='%s', status=~"^4.."}[1m])) / sum(rate(apicast_status{namespace='%s'}[1m])) * 100 > 5`, ns, ns)),
							For:  "5m",
							Labels: map[string]string{
								"severity": "critical",
							},
						},
						{
							Alert: "APICAstLatencyHigh",
							Annotations: map[string]string{
								"summary":     "APICast latency high (instance {{ $labels.instance }})",
								"description": "APIcast p99 latency is higher than 5 seconds\n  VALUE = {{ $value }}\n  LABELS: {{ $labels }}",
							},
							Expr: intstr.FromString(fmt.Sprintf(`histogram_quantile(0.99, sum(rate(total_response_time_seconds_bucket{namespace='%s',}[30m])) by (le)) > 5`, ns)),
							For:  "5m",
							Labels: map[string]string{
								"severity": "warning",
							},
						},
					},
				},
			},
		},
	}
}
