package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	pb "github.com/rajeshreddyt/GrpcServerSubtract/SUBTRACT"
)

// const (
//
//	// primaryReplicaAddr = "192.168.64.3:50050" // Address of the primary replica
//	primaryReplicaAddr = "localhost:50050" // Address of the primary replica
//
// )
var workerPort string
var PodName string

type server struct {
	pb.UnimplementedPBFTServiceServer
}

type replica struct {
	ReplicaId        int
	ReplicaAddress   string
	ReplicaKeyInEtcd string
}

func (s *server) SubtractMethod(ctx context.Context, req *pb.SubtractRequest) (*pb.SubtractResponse, error) {
	// implement your logic here
	replicas, err := getReplicasAddressBasedonMyPodName(PodName)
	if err != nil {
		log.Println("Failed to get replicas address: %v", err)
		return nil, err
	}
	var primaryReplicaAddr string
	fmt.Println("Replicas: ", replicas)
	for _, replica := range replicas {
		fmt.Println("Replica: ", replica)
		if replica.ReplicaId == 0 {
			primaryReplicaAddr = replica.ReplicaAddress
		}
	}
	fmt.Println("Connecting to primary replica...:", primaryReplicaAddr)
	conn, err := grpc.Dial(primaryReplicaAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect to primary replica: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to primary replica")
	// Create a new PBFTService client
	client := pb.NewPBFTServiceClient(conn)

	// Send a computation request to the primary replica
	// request := &pb.ComputationRequest{
	// 	Operand1:  float64(req.A),
	// 	Operand2:  float64(req.B),
	// 	Operation: pb.ComputationRequest_SUBTRACT,
	// }

	request := &pb.ComputationRequest{
		Operand1:  5,
		Operand2:  5,
		Operation: pb.ComputationRequest_SUBTRACT,
	}

	fmt.Printf("Sending computation request: %+v\n", request)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	response, err := client.ProcessComputation(ctx, request)
	if err != nil {
		log.Fatalf("Failed to send computation request: %v", err)
	}

	fmt.Printf("Received computation response: %+v\n", response)

	result := req.GetA() * req.GetB()
	return &pb.SubtractResponse{Result: result}, nil
}

// plain add method
// func (s *server) AddMethod(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error) {
// 	fmt.Println("addmethod call")
// 	fmt.Println("params:", req)
// 	result := req.GetA() + req.GetB()
// 	return &pb.AddResponse{Result: result}, nil
// }

func (s *server) Watch(ctx context.Context, req *pb.HealthCheckRequest) *pb.HealthCheckResponse {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}
}

func (s *server) Check(ctx context.Context, req *pb.HealthCheckRequest) *pb.HealthCheckResponse {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}
}

func main() {

	fmt.Println("================================================with deploymentsv1================================================================")
	ipAddress, err := getPodIPAddress()
	PodDetails := getPodDetails()
	if err != nil {
		panic("ipaddress_err->:" + err.Error())
	}

	fmt.Println("----Pod IP address:", ipAddress)
	if len(PodDetails.Spec.Containers) > 0 {
		workerPort = ":" + strconv.Itoa(int(PodDetails.Spec.Containers[0].Ports[0].ContainerPort))
	}

	fmt.Println("Worker IP Address is", workerPort)

	fmt.Println("================================================================calling others================================================================")

	// Set up a connection to the primary replica
	fmt.Println("================================================================registering  my self into etcd================================================================")
	_ = updateEtcdKeyValue(ipAddress)
	fmt.Println("registration done in etcd with this Worker IP Address =>%s:%s", ipAddress, workerPort)

	fmt.Println("================================================================starting grpc server================================================================")
	// Start the gRPC server
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPBFTServiceServer(s, &server{})
	reflection.Register(s)
	fmt.Printf("Starting worker rpc client on replica lates---- %s \n", ":3000")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func getPodIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("unable to determine pod IP address")
}

func getPodDetails() *corev1.Pod {
	// Load Kubernetes configuration from within the pod
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Get the pod's namespace and name
	namespace := getNamespace()
	podName := getPodName()
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	podIP := pod.Status.PodIP
	// Print the pod IP
	fmt.Println("Pod IP:", podIP)
	// Access the desired pod details

	if strings.Contains(podName, "add-worker") {
		fmt.Println("contains word add worker in podname:>", podName)
		podName = truncateAfterWord(podName, "add-worker")
		fmt.Println("updated and truncated podname:>", podName)
	}
	PodName = podName
	log.Println("================================================globalpod name====>:", PodName)
	// Retrieve the pod details

	// ... access other pod details as needed

	return pod

}

// Helper function to retrieve the pod's namespace
func getNamespace() string {
	nsPath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	nsBytes, err := ioutil.ReadFile(nsPath)
	if err != nil {
		panic(err.Error())
	}
	return string(nsBytes)
}

// Helper function to retrieve the pod's name
func getPodName() string {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Println("POD_NAME not found")
		podName = os.Getenv("HOSTNAME")
		fmt.Println("hostname:>", podName)
		if podName == "" {
			panic("POD_NAME environment variable not set")
		}
	}
	fmt.Println("returning the podname as =>:", podName)
	return podName
}

func getEtcdDetails() (*corev1.Service, error) {
	// Load Kubernetes configuration from within the pod
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get the etcd service details
	serviceName := "etcd" // Replace with the actual name of the etcd service in your cluster
	namespace := getNamespace()

	service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func updateEtcdKeyValue(ipaddress string) error {
	// Load Kubernetes configuration from within the pod
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Get the pod's namespace and name
	namespace := getNamespace()
	podName := getPodName()
	// if podname contains the "add-worker"

	// Retrieve the pod details
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if strings.Contains(podName, "add-worker") {
		fmt.Println("contains word add worker in podname:>", podName)
		podName = truncateAfterWord(podName, "add-worker")
		fmt.Println("updated and truncated podname:>", podName)
	}

	// Retrieve the etcd service details
	etcdDetailService, err := getEtcdDetails()
	if err != nil {
		return err
	}
	etcdIP := etcdDetailService.Spec.ClusterIP
	etcdPort := etcdDetailService.Spec.Ports[0].Port

	fmt.Println("etcdip and etcd port: ", etcdIP, etcdPort)
	// Connect to etcd
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{fmt.Sprintf("%s:%d", etcdIP, etcdPort)},
	})

	if err != nil {
		return err
	}

	defer etcdClient.Close()

	// Retrieve all keys
	getResponse, err := etcdClient.Get(context.TODO(), "", clientv3.WithPrefix())
	if err != nil {
		return err
	}
	fmt.Println("got response from the etcd server getresponse value===>:", getResponse)
	foundKey := ""
	for _, kv := range getResponse.Kvs {
		key := string(kv.Key)
		fmt.Println("key in the etcd responses===>", key)
		if stringContains(key, podName) {
			// Partial match found
			foundKey = key
			break
		}
	}

	if foundKey != "" {
		// Key exists, retrieve the current value
		getResponse, err := etcdClient.Get(context.TODO(), foundKey)
		if err != nil {
			return err
		}
		currentValue := string(getResponse.Kvs[0].Value)
		fmt.Println("Current value of the key %s is %s", foundKey, currentValue)
	} else {
		fmt.Println("Key not found.")
		foundKey = "pods/set-1_add-worker"
		fmt.Println("found key hard coding pods/set-1_add-worker==>", foundKey)
	}

	// Update key-value pair in etcd
	fmt.Println("pod ip from the pod details==>", pod.Status.PodIP)
	fmt.Println("pod ip from param=-==>", ipaddress)
	value := ipaddress + ":" + strconv.Itoa(int(pod.Spec.Containers[0].Ports[0].ContainerPort))
	// key := fmt.Sprintf("%s", foundKey)
	fmt.Println("updating the key in etcd with this new value %s ==> %s", foundKey, value)

	_, err = etcdClient.Put(context.TODO(), foundKey, value)
	if err != nil {
		return err
	}
	// time.Sleep(1 * time.Second)
	fmt.Println("================================================================Updated key-value pair in etcd========")
	crosscheck_value, err := etcdClient.Get(context.TODO(), foundKey)
	if err != nil {
		return err
	}
	crs_value := string(crosscheck_value.Kvs[0].Value)
	fmt.Println("crosscheck_value", crs_value)
	fmt.Println("crosscheck_value", crosscheck_value)
	return nil
}

// Helper function to check if a string contains a substring
func stringContains(s string, substr string) bool {
	return strings.Contains(s, substr)
}

func getReplicasAddressBasedonMyPodName(currentWorkerPodName string) ([]replica, error) {
	fmt.Println("getting replicas addresses by current pod name->:", currentWorkerPodName)
	// Retrieve the etcd service details
	etcdDetailService, err := getEtcdDetails()
	if err != nil {
		return nil, err
	}

	etcdIP := etcdDetailService.Spec.ClusterIP
	etcdPort := etcdDetailService.Spec.Ports[0].Port
	fmt.Println("etcdip and etcd port: ", etcdIP, etcdPort)
	// Connect to etcd
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{fmt.Sprintf("%s:%d", etcdIP, etcdPort)},
	})
	if err != nil {
		return nil, err
	}
	defer etcdClient.Close()

	// Retrieve all keys
	getResponse, err := etcdClient.Get(context.TODO(), "", clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	fmt.Println("got response from the etcd server getresponse value===>:", getResponse)
	// Iterate over all keys
	workerCompleteKey := ""

	for _, kv := range getResponse.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)
		if stringContains(key, "pods") {
			if stringContains(value, currentWorkerPodName) {
				// Partial match found
				workerCompleteKey = key
				break
			}
		}
	}
	fmt.Println("worker complete key: ", workerCompleteKey)

	fmt.Println("================================================using static cluster ip================================================")
	config := clientv3.Config{
		Endpoints: []string{"192.168.59.100:30000"}, // Replace with your etcd server address
	}

	// Create a new client with the configuration
	client, err := clientv3.New(config)
	if err != nil {
		log.Println("Error creating")
		log.Fatal(err)
	}
	defer client.Close()

	log.Println("client created================================")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Get(ctx, "set")
	allkeyValues, err := client.Get(ctx, "", clientv3.WithPrefix())

	fmt.Println("resresp, err := client.Get(ctx, set)", resp)
	fmt.Println("cancel-->", cancel)
	if err != nil {
		log.Fatal(err)
	}
	for _, ev := range allkeyValues.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
		key := string(ev.Key)
		value := string(ev.Value)
		if stringContains(key, "pods") {
			if stringContains(key, currentWorkerPodName) {
				// Partial match found
				workerCompleteKey = key
				fmt.Println("value-->:", value)
				fmt.Println("worker complete key found--------------------------------: ", workerCompleteKey)
				break
			}
		}
	}
	fmt.Println("worker complete key: ", workerCompleteKey)
	fmt.Println("================================================end of using static cluster ip================================================")
	if workerCompleteKey != "" {
		fmt.Println("worker complete key: ", workerCompleteKey)
		// split the key with - delimiter
		// workerCompleteKeySplit := strings.Split(workerCompleteKey, "-")
		// // get the first element of the array
		// setname := workerCompleteKeySplit[0]

		// based on the setname concatenated with AddReplica0 which is our primary
		// get the value of the key
		fmt.Println("================================================================getting replica addresses================================================================")
		var replicas []replica
		rid := 0
		for _, ev := range allkeyValues.Kvs {
			fmt.Printf("%s : %s\n", ev.Key, ev.Value)
			key := string(ev.Key)
			value := string(ev.Value)
			if stringContains(key, "replica") {
				replica := replica{
					ReplicaAddress:   value,
					ReplicaId:        rid,
					ReplicaKeyInEtcd: key,
				}
				replicas = append(replicas, replica)
				rid++
			}
		}
		// for i := 0; i <= 4; i++ {
		// 	// value, err := client.Get(context.TODO(), setname+"_add-replica-"+strconv.Itoa(i),
		// 	// 	clientv3.WithPrefix())
		// 	// if err != nil {
		// 	// 	return nil, err
		// 	// }
		// 	// fmt.Println(setname + "-add-replica-" + strconv.Itoa(i))
		// 	// fmt.Println("replica value from get===>:", value)

		// 	// push the

		// }
		fmt.Println("returning replicas: ", replicas)
		return replicas, nil
	} else {
		return nil, nil
	}

}

func truncateAfterWord(str, word string) string {
	index := strings.Index(str, word)
	if index != -1 {
		return str[:index+len(word)]
	}
	return str
}
