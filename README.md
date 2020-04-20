# GraphQL Mesh Operator

Aggregate APIs inside Kubernetes and expose them as GraphQL API outside Kubernetes with [GraphQL Mesh](https://github.com/Urigo/graphql-mesh).

Apr/2020, Only design. Not complete. 

## Design

- Example: kubernetes api server
    - Create namespace graphql-mesh
    - Set `default` serviceaccount as cluster-admin
    - Mount secret `default-token` to pod `gateway`
    - Inside a pod `gateway`, run [GraphQL Mesh](https://github.com/Urigo/graphql-mesh)
    - Configure `.meshrc.yaml` to set bearer token
    - Expose GraphQL Mesh endpoints to outside Kuberntes

## Interface

You can add new GraphQL mesh instance on Kubernetes with this resource.

```yaml
apiVersion: graphql-mesh-operator.io
kind: GraphqlMesh
metadata:
  name: sample-gateaway
spec:
  meshrc:
    sources:
    - name: Wiki
      handler:
      openapi:
        source: https://api.apis.guru/v2/specs/wikimedia.org/1.0.0/swagger.yaml
  meshrcConfigMap:
    configMapName: test-configmap
  meshrcSecret:
    secretName: test-secret
```

## Build

```sh
make install
make run
```