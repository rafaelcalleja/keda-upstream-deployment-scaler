package main

import (
	"context"
	"fmt"
	pb "github.com/rafaelcalleja/keda-upstream-deployment-scaler/externalscaler"
	"log"
	"net"
	"os"
	"strconv"

	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	discovery "github.com/gkarthiks/k8s-discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type ExternalScaler struct{}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	namespace := scaledObject.ScalerMetadata["upstreamDeploymentNamespace"]
	deployment := scaledObject.ScalerMetadata["upstreamDeploymentName"]

	if len(namespace) == 0 {
		return nil, status.Error(codes.InvalidArgument, "upstreamDeploymentNamespace must be specified")
	}

	if len(deployment) == 0 {
		return nil, status.Error(codes.InvalidArgument, "upstreamDeploymentName must be specified")
	}

	options, err := kubeClient.AppsV1().Deployments(namespace).GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return &pb.IsActiveResponse{
			Result: false,
		}, err
	}

	return &pb.IsActiveResponse{
		Result: options.Spec.Replicas > 0,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "targetSize",
			TargetSize: 1,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	replicaCount := metricRequest.ScaledObjectRef.ScalerMetadata["replicaCount"]

	if len(replicaCount) == 0 {
		return nil, status.Error(codes.InvalidArgument, "replicaCount must be specified")
	}

	replicaCountInt64, err := strconv.ParseInt(replicaCount, 10, 64)
	if nil != err {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "targetSize",
			MetricValue: replicaCountInt64,
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	namespace := scaledObject.ScalerMetadata["upstreamDeploymentNamespace"]
	deployment := scaledObject.ScalerMetadata["upstreamDeploymentName"]

	if len(namespace) == 0 {
		return status.Error(codes.InvalidArgument, "upstreamDeploymentNamespace must be specified")
	}

	if len(deployment) == 0 {
		return status.Error(codes.InvalidArgument, "upstreamDeploymentName must be specified")
	}

	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Minute * 10):
			options, err := kubeClient.AppsV1().Deployments(namespace).GetScale(context.Background(), deployment, metav1.GetOptions{})

			if err != nil {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: false,
				})
			} else if options.Spec.Replicas > 0 {
				err = epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
			}
		}
	}
}

var kubeClient *kubernetes.Clientset

func main() {
	k8sCli, err := discovery.NewK8s()

	if err != nil {
		log.Fatal(err, "discovering new Kubernetes config")
		os.Exit(1)
	}

	kubeClient, err = kubernetes.NewForConfig(k8sCli.RestConfig)
	if err != nil {
		log.Fatal(err, "creating new Kubernetes ClientSet")
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":9001")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting on :9001")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
