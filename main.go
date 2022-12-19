package main

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	client          kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	mapper          meta.RESTMapper
	dynamicClient   dynamic.Interface
	config          *rest.Config
)

var deploymentManifest = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name": "demo-deployment",
		},
		"spec": map[string]interface{}{
			"replicas": 2,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": "demo",
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": "demo",
					},
				},

				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "web",
							"image": "nginx:1.12",
							"ports": []map[string]interface{}{
								{
									"name":          "http",
									"protocol":      "TCP",
									"containerPort": 80,
								},
							},
						},
					},
				},
			},
		},
	},
}

type Dynamic struct {
	Client    dynamic.Interface
	Discovery discovery.DiscoveryInterface
	Mapper    meta.RESTMapper
}

func (d Dynamic) NewClient(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	// dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	// _, gvk, err := dec.Decode(data, nil, obj)
	// fmt.Printf("%v\n", obj)
	gvk := obj.GroupVersionKind()
	// if err != nil {
	// 	return nil, err
	// }

	mapping, err := d.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(metav1.NamespaceDefault)
		}
		return d.Client.Resource(mapping.Resource).Namespace(obj.GetNamespace()), nil
	} else {
		return d.Client.Resource(mapping.Resource), nil
	}
}

func (d Dynamic) Apply(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	fmt.Printf("Apply start... %v\n", obj)
	data, err := json.Marshal(obj)
	if err != nil {
		fmt.Printf("Apply error")
		return nil, err
	}
	fmt.Printf("Apply interal\n")

	return client.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "example",
	})
}

func InitK8s() error {
	conf, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}
	config = conf

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}
	client = clientset

	discoveryClient = clientset.Discovery()

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return err
	}
	mapper = restmapper.NewDiscoveryRESTMapper(groupResources)

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	dynamicClient = dyn

	return nil
}

func main() {
	InitK8s()

	// pods, err := client.CoreV1().Pods("sample").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	// for _, p := range pods.Items {
	// 	fmt.Printf("- %s\n", p.ObjectMeta.Name)
	// }

	dynamic := Dynamic{
		Client:    dynamicClient,
		Discovery: discoveryClient,
		Mapper:    mapper,
	}
	// obj = &unstructured.Unstructured{}
	cl, err := dynamic.NewClient(deploymentManifest)
	if err != nil {
		return
	}
	res, err := dynamic.Apply(context.TODO(), cl, deploymentManifest)
	fmt.Printf("applied %s %s %s", res.GroupVersionKind(), res.GetNamespace(), res.GetName())
}

func testJson() {
	type GoStruct struct {
		A int
		B string
	}
	stcData := GoStruct{A: 1, B: "bbb"}

	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(jsonData)
}
