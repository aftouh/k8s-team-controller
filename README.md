# k8s-teams-controller

`k8s-teams-controller` is a kubernetes controller for the `teams.aftouh.io` CRD.  
Team is a cluster scoped resource that manages team's namespace and resourcequota.

<img src="./logo/kubernetes.svg" width="100">

```yaml
apiVersion: aftouh.io/v1
kind: Team
metadata:
  name: poc-dev
spec:
  name: poc
  environment: dev
  description: "poc team is creating a product ..."
  resourceQuota:
    hard:
      pods: "4"
```

## Motivation

This project is created to build a sample of a kubernetes controller and understand what's under the hood.  
This kubernetes [controller sample](https://github.com/kubernetes/sample-controller) helped me so much 🙏.

## Development

### Tools

- [k8s.io/code-generator](https://github.com/kubernetes/code-generator): generate deepcopy, clientset, informers and listers of the team CRD
- [ko](https://github.com/google/ko): build and deploy controller in the kubernetes cluster

### Run teams controller

Run controller locally:

```bash
go run ./cmd/controller -kubeconfig ~/.kube/config -v 5
```

Run within a kubernetes cluster using `ko`:

```bash
# login to dockerhub
docker login
export KO_DOCKER_REPO=ftahmed

# build & deploy using ko
ko apply -f config/
```

### Generate code

Command for generating deepcopy, clientset, infromers and listers of the team resource

```bash
go mod vendor
./hack/update-codegen.sh
```

### Run tests

```bash
go test -v ./...
```
