# How to get the CA cert
```
kubectl get secret quickstart-es-http-ca-internal -o jsonpath="{.data.tls\.crt}" | base64 -d > ca.crt
```

# Steps I carried out:
## Step 0: Prerequisites
Your Elasticsearch cluster is created via ECK.

You have cert-manager installed in the cluster.

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml


## Step 1: Extract the ECK CA Cert
Get the CA from the ECK-generated secret:

kubectl get secret quickstart-es-http-certs-internal -o jsonpath='{.data.ca\.crt}' | base64 -d > eck-ca.crt

## Step 2: Create a Secret for the CA

# eck-ca-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: eck-ca-secret
  namespace: default
type: Opaque
data:
  ca.crt: <base64-encoded-ca.crt>

```
kubectl apply -f eck-ca-secret.yaml

```
## Step 3: Create a cert-manager Issuer
# eck-client-cert-issuer.yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: eck-ca-issuer
  namespace: default
spec:
  ca:
    secretName: eck-ca-secret

```
kubectl apply -f eck-client-cert-issuer.yaml
```

## Step 4: Create a Certificate Resource
# eck-client-cert.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: elasticsearch-client-cert
  namespace: default
spec:
  secretName: elasticsearch-client-cert-secret
  commonName: my-golang-client
  duration: 2160h  # 90 days
  renewBefore: 360h # 15 days before expiry
  usages:
    - client auth
  issuerRef:
    name: eck-ca-issuer
    kind: Issuer
  privateKey:
    algorithm: RSA
    size: 2048

```
kubectl apply -f eck-client-cert.yaml
```
## Step 5: Get Your Cert and Key
```
kubectl get secret elasticsearch-client-cert-secret -o jsonpath='{.data.tls\.crt}' | base64 -d > client.crt
kubectl get secret elasticsearch-client-cert-secret -o jsonpath='{.data.tls\.key}' | base64 -d > client.key
```

# Problem with the above steps was I didn't have the ca.key as ECK doesn't give that to us.
## base 64 encoded ca.key should have been part of the data field in "eck-ca-secret.yaml"

## Step 1: Create your own CA:
```
openssl genrsa -out my-root-ca.key 4096
openssl req -x509 -new -nodes -key my-root-ca.key -sha256 -days 3650 -out my-root-ca.crt -subj "/CN=my-root-ca"

```
## Step 2: Create a Kubernetes secret with both ca.crt and ca.key:
# my-ca-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-ca-secret
  namespace: default
type: Opaque
data:
  tls.crt: <base64-encoded my-root-ca.crt>
  tls.key: <base64-encoded my-root-ca.key>

```
kubectl apply -f my-ca-secret.yaml
```

## Step 3 Use it in your Issuer:
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: my-ca-issuer
  namespace: default
spec:
  ca:
    secretName: my-ca-secret

```
kubectl apply -f eck-client-cert-issuer.yaml
```



## Step 4: Create a Certificate Resource
# eck-client-cert.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: elasticsearch-client-cert
  namespace: default
spec:
  secretName: elasticsearch-client-cert-secret
  commonName: my-golang-client
  duration: 2160h  # 90 days
  renewBefore: 360h # 15 days before expiry
  usages:
    - client auth
  issuerRef:
    name: my-ca-issuer
    kind: Issuer
  privateKey:
    algorithm: RSA
    size: 2048

```
kubectl apply -f eck-client-cert.yaml
```
## Step 5: Get Your Cert and Key
```
kubectl get secret elasticsearch-client-cert-secret -o jsonpath='{.data.tls\.crt}' | base64 -d > client.crt
kubectl get secret elasticsearch-client-cert-secret -o jsonpath='{.data.tls\.key}' | base64 -d > client.key
```
## Step 6
spec:
  http:
    tls:
      clientAuth: required
      certificateAuthorities:
        - secretName: my-ca-secret


## Ran into another issue after tha above step.
```
mtls-elasticsearch git:(main) ✗ go run main.go
{"error":{"root_cause":[{"type":"security_exception","reason":"missing authentication credentials for REST request [/_cat/health?v]","header":{"WWW-Authenticate":["Basic realm=\"security\" charset=\"UTF-8\"","Bearer realm=\"security\"","ApiKey"]}}],"type":"security_exception","reason":"missing authentication credentials for REST request [/_cat/health?v]","header":{"WWW-Authenticate":["Basic realm=\"security\" charset=\"UTF-8\"","Bearer realm=\"security\"","ApiKey"]}},"status":401}
```

## I didn't explore mTLS beyond this step because we decided we will proceed with API Key
## further steps are:
Even though you’re using mTLS, Elasticsearch doesn't know who your client cert belongs to, unless you explicitly tell it to map that cert’s distinguished name (DN) to a user.
By default, ECK-generated Elasticsearch:
Requires HTTPS (via TLS)
Does not use client certs as authentication
Expects Basic Auth or other configured methods unless you configure PKI Realm

Fix:
## Step 1: Fix: Enable PKI Authentication in Elasticsearch (via ECK)
Create a PKI realm by adding it to the spec.nodeSets[].config section in your Elasticsearch manifest:

```
spec:
  nodeSets:
    - name: default
      count: 1
      config:
        node.roles: ["master", "data", "ingest"]
        xpack.security.authc.realms.pki.pki1.order: 0
        xpack.security.authc.realms.pki.pki1.enabled: true
        xpack.security.authc.realms.pki.pki1.certificate_authorities: [ "/usr/share/elasticsearch/config/ca.crt" ]
        xpack.security.authc.realms.pki.pki1.username_pattern: "CN=(.*?)(?:,|$)"
```
## Step 2:
Mount the CA certificate used to sign your client cert (the one used in ca.crt) into the Elasticsearch pod:
Create a Kubernetes secret containing your ca.crt:
```
kubectl create secret generic my-pki-ca-secret --from-file=ca.crt
```
Then mount it on the Elasticsearch Pod
```
spec:
  nodeSets:
    - name: default
      ...
      podTemplate:
        spec:
          containers:
            - name: elasticsearch
              volumeMounts:
                - name: pki-ca
                  mountPath: /usr/share/elasticsearch/config/ca.crt
                  subPath: ca.crt
          volumes:
            - name: pki-ca
              secret:
                secretName: my-pki-ca-secret
```
## Step 3:
Create a role mapping so your certificate's subject (e.g., CN=dev-client) is mapped to a role:

```
kubectl exec -it quickstart-es-default-0 -- bash
```
Then inside the pod:

bin/elasticsearch-users roles dev-role

Or via API:

curl -X POST https://<elasticsearch-url>/_security/role_mapping/pki1 \
  -u elastic:your-password \
  -H "Content-Type: application/json" \
  -d '{
    "roles": [ "superuser" ],
    "enabled": true,
    "rules": { "field": { "dn": "CN=dev-client" } }
  }'

Replace "CN=dev-client" with the subject DN of your client certificate (openssl x509 -in client.crt -noout -subject).

