## Development

Developing on the kubernetes provider is supported via the [skaffold](https://github.com/GoogleContainerTools/skaffold) tool.

A simple `skaffold.yaml` file is included in the root of this project.

You can see details about using `skaffold` [here](https://skaffold-latest.firebaseapp.com/)

You will also need access to either a remote or local kubernetes cluster.

You can get a local cluster via either [minikube](https://kubernetes.io/docs/setup/minikube/) or
[docker-for-desktop](https://docs.docker.com/docker-for-mac/kubernetes/)

By default the provided configuration deploys each build as a statefulset to the `eco` namespaces,
skaffold will automatically create the namespace if it doesn't exist in the cluster.

To get started, on Mac OS X run the following commands from the project root:

```
brew install skaffold
skaffold dev
```

