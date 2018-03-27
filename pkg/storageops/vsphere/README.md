### To test vSphere

You will first need to create a GCE instance and then provide details of this instance as below.

```bash
export GOOGLE_APPLICATION_CREDENTIALS=<path-to-service-account-json-file>
export GCE_INSTANCE_NAME=<gce-instance-name>
export GCE_INSTANCE_ZONE=<gce-instance-zone>
export GCE_INSTANCE_PROJECT=<gce-project-name>

go test
```


