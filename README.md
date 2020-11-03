# Ploy - straightforward Kubernetes deployment

Ploy is a proof of concept tool that will deploy a Docker image on your machine locally to a Kubernetes cluster.

It was inspired by [Halloumi](https://github.com/pulumi/halloumi) and its purpose is to show how easy it is to create useful developer tooling with the [Pulumi](https://pulumi.com/) [Automation API](https://pkg.go.dev/github.com/pulumi/pulumi/sdk/v2/go/x/auto)

It provides a Heroku like experience for uses deploying Docker images.

It is currently designed to work exclusively with AWS EKS, but support for other providers could be added.

## Installation



## Usage

Ploy take a Docker context with a `Dockerfile`, builds it locally and pushes it to an [ECR repository](https://docs.aws.amazon.com/AmazonECR/latest/userguide/Repositories.html).

Each ploy application is deployed as a pulumi stack within a project called `ploy` in your configured Pulumi organization (see [configuration](##configuration))

Once the image is pushed, it creates a Kubernetes namespace, Deployment and Service with an external load balancer.

Here's what it looks like:

### Deploy

```bash
ploy up
INFO[0002] Creating application: regularly-viable-stud
INFO[0002] Creating ECR repository
INFO[0002] Creating local docker image
INFO[0002] Creating Kubernetes namespace
INFO[0002] Creating Kubernetes deployment
INFO[0002] Creating Kubernetes service
INFO[0004] Repository created: 616138583583.dkr.ecr.us-west-2.amazonaws.com/regularly-viable-stud-e23717a
INFO[0034] Your service is available at: aedc7304ebcf648baa881cf4069e5aad-354579065.us-west-2.elb.amazonaws.com
```

Ploy deploys your application to the Kubernetes cluster currently configured in your `KUBECONFIG`. It creates the ECR repository using the AWS credentials you're currently using, whether that be an aws profile or aws keys.

_note:_ Your EKS cluster must have access to ECR for the image to be pulled. See [here](https://docs.aws.amazon.com/AmazonECR/latest/userguide/ECR_on_EKS.html) for more details.

You can optionally set an explicit name for your application by passing it as an argument:

```bash
ploy up my-app
```

### Retrieve

You can grab a list of the currently deployed ploy applications using the `get` command:

```bash
ploy get
+--------------------------+--------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------+
|           NAME           |       LAST UPDATE        |                        DEPLOYMENT INFO                         |                                      URL                                      |
+--------------------------+--------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------+
| frequently-better-beagle | 2020-11-03T19:06:36.000Z | https://app.pulumi.com/jaxxstorm/ploy/frequently-better-beagle | http://a9027e3626d3a41a78e600ab832991d3-975080570.us-west-2.elb.amazonaws.com |
| regularly-viable-stud    | 2020-11-03T19:41:19.000Z | https://app.pulumi.com/jaxxstorm/ploy/regularly-viable-stud    | http://aedc7304ebcf648baa881cf4069e5aad-354579065.us-west-2.elb.amazonaws.com |
+--------------------------+--------------------------+----------------------------------------------------------------+-------------------------------------------------------------------------------+
```

### Destroy

You can tear down your `ploy` application with the `destroy` command:

```bash
ploy destroy
```

## Configuration

Ploy's only required configuration value is your Pulumi org. You can specify it on the command line:

### Organization

```bash
ploy up -o jaxxstorm
```

Or alternatively, set it in your ploy configuration file:

```yaml
cat ~/.ploy/config.yml
org: jaxxstorm
```

### Region

You'll need to set the AWS region you want to use for your ECR repository. You can set it on the command line:

```bash
ploy up -r us-west-2
```

Via configuration:

```yaml
cat ~/.ploy/config.yml
region: us-west-2
```

Or alternatively, it'll read your `AWS_REGION` environment variable:

```
export AWS_REGION=us-west-2
```
