# Every elasticsearch service has a username "elastic".

## Portforward
```
kubectl port-forward service/quickstart-es-http 9200
```
## Command I used to get the password:
```
 kubectl get secret quickstart-es-elastic-user -o go-template='{{.data.elastic | base64decode}}'
 ```

## Command to use the password:
```
The -k flag (or --insecure) tells curl to not validate the SSL certificate when connecting over HTTPS.

curl -u elastic:<password> https://localhost:9200 -k
```

## Command to get the ca.crt
```
kubectl get secret quickstart-es-http-certs-internal -o jsonpath='{.data.ca\.crt}' | base64 -d > eck-ca.crt
```

# When I used CURL with ca certificate I got an error.
```
curl --cacert eck-ca.crt -u elastic:ZQP7Mr66F76Uz7X6G0c6XRA8 https://localhost:9200

curl: (60) SSL: no alternative certificate subject name matches target host name 'localhost'
More details here: https://curl.se/docs/sslcerts.html

curl failed to verify the legitimacy of the server and therefore could not
establish a secure connection to it. To learn more about this situation and
how to fix it, please visit the web page mentioned above.
```
Next what I ended up doing is adding this to /etc/localhost
127.0.0.1 quickstart-es-http.default.svc

Because the TLS cert is for the quickstart-es-http.default.svc

##
What Elasticsearch provides by default:
When TLS is enabled (which it is by default with the ECK operator for HTTP), Elasticsearch generates:

A CA certificate (ca.crt) – the Certificate Authority used to sign the server's certificate.

A server certificate and key (tls.crt and tls.key) – these are used by Elasticsearch to serve HTTPS traffic.

These are typically bundled in a Kubernetes Secret like quickstart-es-http-certs.

##
TLS Certificate vs CA Certificate
1. TLS Certificate (Server Certificate)
Created by ECK for the Elasticsearch HTTP service.
Used by the Elasticsearch server to prove its identity during TLS handshakes.
It contains:
Subject (e.g., CN=quickstart-es-http.default.svc)
SANs (Subject Alternative Names), like:
quickstart-es-http.default.svc
quickstart-es-http.default.svc.cluster.local
Does not usually include localhost unless We explicitly configure it.

2. CA Certificate (ca.crt)
The certificate authority (CA) that signed the server certificate.
Used by clients to verify that the server certificate is trustworthy.
It's not tied to any hostname.
We can have the ca.crt and still get hostname mismatch errors if the TLS certificate itself doesn't include the host We're trying to connect to (like localhost).

So when curl complains:
SSL: no alternative certificate subject name matches target host name 'localhost'
It’s checking the server’s TLS certificate, not the ca.crt, and saying:
"The certificate I got from the server doesn’t claim to be for localhost, so I don’t trust it."

