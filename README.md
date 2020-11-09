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

## Interface `GraphqlMesh`

You can add new GraphQL mesh instance on Kubernetes with this resource.
`GraphqlMesh` is a resource for definition of an entrypoints of GraphQL Mesh.
When you define one `GraphqlMesh`, this operator creates one GraphQL Mesh server.

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

`meshrc` value should be the same as original GraphQL Mesh `.meshrc.yaml` file described [here](https://graphql-mesh.com/docs/getting-started/basic-example/).

## Advanced interface `DataSource`

In this advanced usage, you can organize `meshrc` automatically.

`GraphqlMesh` resource can include many separate `DataSource` resources, which defines data source to GraphQL Mesh server. And this `DataSource` is reusable in many GraphQL Mesh servers.

```yaml
apiVersion: mesh.graphql-mesh-operator.io/v1alpha1
kind: GraphqlMesh
metadata:
  name: sample
spec:
  dataSourceNames:
  - wiki
```

```yaml
apiVersion: mesh.graphql-mesh-operator.io/v1alpha1
kind: DataSource
metadata:
  name: wiki
spec:
  type: openapi
  handlerConfig:
    source:  https://api.apis.guru/v2/specs/wikimedia.org/1.0.0/swagger.yaml
```

Additionally, `GraphqlMesh` itself can be `DataSource` as well. You can set `asNewDataSource` flag in `GraphqlMesh`.

```yaml
Version: mesh.graphql-mesh-operator.io/v1alpha1
kind: GraphqlMesh
metadata:
  name: sample
spec:
  dataSourceNames:
  - wiki
  asNewDataSource: true
```

Then you can use this `DataSource` name in new `GraphqlMesh` like this.

```yaml
Version: mesh.graphql-mesh-operator.io/v1alpha1
kind: GraphqlMesh
metadata:
  name: sample-wrapper
spec:
  dataSourceNames:
  - sample
```

## Installation

Simply run on this kubernetes environment.

```sh
kubectl apply -f https://raw.githubusercontent.com/onelittlenightmusic/graphql-mesh-operator/master/install.yaml
```

## Build and run by yourself

```sh
make install
make run
```