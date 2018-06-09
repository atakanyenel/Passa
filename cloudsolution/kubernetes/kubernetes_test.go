package kubernetes

import (
	"fmt"
	"testing"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetAll(t *testing.T) {

	config, err := clientcmd.BuildConfigFromFlags("", "/Users/atakanyenel/Desktop/ders/research/kubernetes/newconfig.txt")
	clientset, err := kubernetes.NewForConfig(config)
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	fmt.Printf("Listing deployments in namespace %q:\n", apiv1.NamespaceDefault)
	list, err := deploymentsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, d := range list.Items {
		fmt.Printf(" * %s ---> %d \n", d.Spec.Selector, *d.Spec.Replicas)
	}

	nodesList, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, node := range nodesList.Items {
		fmt.Printf(" * %v \n %v ", node.Name, node.Annotations["flannel.alpha.coreos.com/public-ip"])
	}

}
