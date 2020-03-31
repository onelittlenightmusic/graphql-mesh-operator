# graphql-gateway

Aggregate APIs inside Kubernetes and expose them as GraphQL API outside Kubernetes.

## Design

- Example: kubernetes api server
    - create namespace graphql-gateway
    - set `default` serviceaccount as cluster-admin
    - mount secret `default-token` to pod `gateway`
    - inside a pod `gateway`, run [GraphQL Mesh (https://github.com/Urigo/graphql-mesh)
    - configure `.meshrc.yaml` to set bearer token
    - expose GraphQL Mesh 


## Interface

```yaml
apiVersion: graphqlgw.io
kind: GraphqlGateway
metadata:
  name: sample-gateaway
spec:
  meshrc:
    sources:
    - name: Wiki
      handler:
      openapi:
        source: https://api.apis.guru/v2/specs/wikimedia.org/1.0.0/swagger.yaml
```