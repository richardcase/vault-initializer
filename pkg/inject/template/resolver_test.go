package template

import (
	"fmt"
	"testing"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testPathMissingToken = "/v1/secret/{{.MissingToken}}"
	testPathBad          = "/v1/secret/{{.Namespace}/{{.ContainerName}"
)

func TestResolvePath(t *testing.T) {
	testCases := []struct {
		namespace      string
		deploymentname string
		containername  string
		template       string
		expected       string
	}{
		{"myns", "depname", "contname", "/v1/secret/{{.Namespace}}/{{.ContainerName}}", "/v1/secret/myns/contname"},
		{"myns", "depname", "contname", "/v1/secret/{{.Namespace}}/{{.DeploymentName}}", "/v1/secret/myns/depname"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resolving with %s, %s, %s", tc.namespace, tc.deploymentname, tc.containername), func(t *testing.T) {
			fakeDeployment := deployment(tc.namespace, tc.deploymentname, tc.containername)
			actualPath, err := ResolveTemplate(&fakeDeployment, tc.template)
			if err != nil {
				t.Errorf("Unexpected error resolving template: %v", err)
			}
			if actualPath != tc.expected {
				t.Errorf("Unexpected path resolved: %s, but expected: %s", actualPath, tc.expected)
			}
		})
	}
}

func TestResolvePathErrorOnBadPath(t *testing.T) {
	fakeDeployment := deployment("myns", "depname", "contname")
	_, err := ResolveTemplate(&fakeDeployment, testPathBad)
	if err == nil {
		t.Error("Expected to receive an error becauase of bad path but no error received")
	}
}

func TestResolvePathErrorOnMissingToken(t *testing.T) {
	fakeDeployment := deployment("myns", "depname", "contname")
	_, err := ResolveTemplate(&fakeDeployment, testPathMissingToken)
	if err == nil {
		t.Error("Expected to receive an error becauase of missing token value but no error received")
	}
}

func deployment(namespace string, deploymentname string, containername string) v1beta1.Deployment {
	container := v1.Container{Name: containername}

	return v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      deploymentname,
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{container},
				},
			},
		},
	}
}
