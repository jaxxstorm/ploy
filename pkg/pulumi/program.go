package pulumi

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ecr"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	log "github.com/sirupsen/logrus"
)

type PloyDeployment struct {
	pulumi.ResourceState
	PloyDeploymentArgs PloyDeploymentArgs  `pulumi:"PloyDeploymentArgs"`
	ImageName          pulumi.StringOutput `pulumi:"ImageName"`
}

type PloyDeploymentArgs struct {
	Directory string
	Nlb       bool
}

func NewPloyDeployment(ctx *pulumi.Context, name string, args *PloyDeploymentArgs, opts ...pulumi.ResourceOption) (*PloyDeployment, error) {
	ployDeployment := &PloyDeployment{}

	// register a component resource to group all the resource together
	err := ctx.RegisterComponentResource("ploy:index:Deployment", name, ployDeployment, opts...)
	if err != nil {
		return nil, err
	}

	repo, err := ecr.NewRepository(ctx, name, &ecr.RepositoryArgs{})
	if err != nil {
		return nil, err
	}

	// retrieve the credentials from the ECR repo
	repoCreds := repo.RegistryId.ApplyT(func(id string) ([]string, error) {
		creds, err := ecr.GetCredentials(ctx, &ecr.GetCredentialsArgs{
			RegistryId: id,
		}, pulumi.Parent(ployDeployment))
		if err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(creds.AuthorizationToken)
		if err != nil {
			fmt.Println("error:", err)
			return nil, err
		}

		return strings.Split(string(data), ":"), nil
	}).(pulumi.StringArrayOutput)

	repoUser := repoCreds.Index(pulumi.Int(0))
	repoPass := repoCreds.Index(pulumi.Int(1))

	// build the docker image
	image, err := docker.NewImage(ctx, name, &docker.ImageArgs{
		Build: docker.DockerBuildArgs{
			Context: pulumi.String(filepath.Join(args.Directory)),
		},
		ImageName: pulumi.Sprintf("%s:%d", repo.RepositoryUrl, pulumi.Int(time.Now().Unix())),
		Registry: docker.ImageRegistryArgs{
			Server:   repo.RepositoryUrl,
			Username: repoUser,
			Password: repoPass,
		},
	}, pulumi.Parent(ployDeployment))

	if err != nil {
		return nil, err
	}

	ployDeployment.ImageName = image.ImageName

	// Now we need to handle the Kubernetes of it all
	labels := pulumi.StringMap{
		"app.kubernetes.io/app": pulumi.String(name),
		"app.getploy.io/name":   pulumi.String(name),
	}

	namespace, err := corev1.NewNamespace(ctx, name, &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:   pulumi.String(name),
			Labels: labels,
		},
	}, pulumi.Parent(ployDeployment))
	if err != nil {
		return nil, err
	}

	_, err = appsv1.NewDeployment(ctx, name, &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: namespace.Metadata.Name().Elem(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpecArgs{
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Replicas: pulumi.Int(3),
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:   pulumi.String(name),
					Labels: labels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String("name"),
							Image: image.ImageName,
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(80), // FIXME make this configurable
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(namespace), pulumi.DependsOn([]pulumi.Resource{image}))
	if err != nil {
		return nil, err
	}

	var serviceType pulumi.String
	var annotations pulumi.StringMap
	if args.Nlb {
		serviceType = "NodePort"
		annotations = pulumi.StringMap{
			"service.beta.kubernetes.io/aws-load-balancer-type": pulumi.String("nlb-ip"),
		}
	} else {
		serviceType = "LoadBalancer"
		annotations = pulumi.StringMap{}
	}

	service, err := corev1.NewService(ctx, name, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(name),
			Namespace:   namespace.Metadata.Name().Elem(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: &corev1.ServiceSpecArgs{
			Ports: corev1.ServicePortArray{
				corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(80),
				},
			},
			Type:     serviceType,
			Selector: labels,
		},
	}, pulumi.Parent(namespace), pulumi.DependsOn([]pulumi.Resource{image}))
	if err != nil {
		return nil, err
	}

	ctx.Export("address", service.Status.ApplyT(func(status *corev1.ServiceStatus) *string {
		ingress := status.LoadBalancer.Ingress[0]
		if ingress.Hostname != nil {
			log.Infof("Your service is available at: %v", *ingress.Hostname)
			return ingress.Hostname
		}
		log.Info("Your service is available at: %v", *ingress.Ip)
		return ingress.Ip
	}))

	ctx.RegisterResourceOutputs(ployDeployment, pulumi.Map{
		"ImageName": ployDeployment.ImageName,
	})

	return ployDeployment, nil
}

func Deploy(name string, directory string, nlb bool) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {

		_, err := NewPloyDeployment(ctx, name, &PloyDeploymentArgs{
			Directory: directory,
			Nlb:       nlb,
		})
		if err != nil {
			return err
		}

		return nil
	}

}
