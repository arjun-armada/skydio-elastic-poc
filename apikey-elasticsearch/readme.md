# Another way to authenticate with the Elasticsearch server is via the APIKey. 
## Curl request to get the API Key:

```
curl -k -X POST "https://localhost:9200/_security/api_key" \
  -H "Content-Type: application/json" \
  -u elastic:<Password> \
  -d '{
    "name": "my-app-key",
    "role_descriptors": {
      "my_app_role": {
        "cluster": ["monitor"],
        "index": [
          {
            "names": ["*"],
            "privileges": ["read"]
          }
        ]
      }
    }
  }'
```

## This is what a response looks like:
```
{"id":"SVxlf5YB5SRb69WuvF-X","name":"my-app-key","api_key":"<API Key>","encoded":"<API Key-encoded>"}
```

## Curl request to use the API Key
```
The -k flag (or --insecure) tells curl to not validate the SSL certificate when connecting over HTTPS.


curl -k -H "Authorization: ApiKey <API Key>" https://quickstart-es-http.default.svc:9200
{
  "name" : "quickstart-es-default-0",
  "cluster_name" : "quickstart",
  "cluster_uuid" : "leYXj8sUT4i5Ozr4Weq8bA",
  "version" : {
    "number" : "8.12.2",
    "build_flavor" : "default",
    "build_type" : "docker",
    "build_hash" : "48a287ab9497e852de30327444b0809e55d46466",
    "build_date" : "2024-02-19T10:04:32.774273190Z",
    "build_snapshot" : false,
    "lucene_version" : "9.9.2",
    "minimum_wire_compatibility_version" : "7.17.0",
    "minimum_index_compatibility_version" : "7.0.0"
  },
  "tagline" : "You Know, for Search"
}
```

## Error with localhost
```
curl --cacert eck-ca.crt -H "Authorization: ApiKey WEVyVGtwWUJpa2VaMVprU19EUkk6cDdFQm5GdDFRWHl5VDNaX0ZqSU4wQQ==" https://localhost:9200
curl: (60) SSL: no alternative certificate subject name matches target host name 'localhost'
More details here: https://curl.se/docs/sslcerts.html

curl failed to verify the legitimacy of the server and therefore could not
establish a secure connection to it. To learn more about this situation and
how to fix it, please visit the web page mentioned above.
```

## /etc/hosts
```
~ cat /etc/hosts
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
127.0.0.1 quickstart-es-http.default.svc
```