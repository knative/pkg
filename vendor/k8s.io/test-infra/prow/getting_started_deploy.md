# Deploying Prow

This document will walk you through deploying your own Prow instance to a new Kubernetes cluster. If you encounter difficulties, please open an issue so that we can make this process easier.

Prow runs in any kubernetes cluster. Our `tackle` utility helps deploy it correctly, or you can perform each of the steps manually.

Both of these are focused on [Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) but should work on any kubernetes distro with no/minimal changes.

## GitHub bot account

Before using `tackle` or deploying prow manually, ensure you have created a
GitHub account for prow to use.  Prow will ignore most GitHub events generated
by this account, so it is important this account be separate from any users or
automation you wish to interact with prow. For example, you still need to do
this even if you'd just setting up a prow instance to work against your own
personal repos.

1. Ensure the bot user has the following permissions
  - Write access to the repos you plan on handling
  - Owner access (and org membership) for the orgs you plan on handling (note
    it is possible to handle specific repos in an org without this)
1. Create a [personal access token][1] for the GitHub bot account, adding the
   following scopes (more details [here][8])
  - Must have the `public_repo` and `repo:status` scopes
  - Add the `repo` scope if you plan on handing private repos
  - Add the `admin_org:hook` scope if you plan on handling a github org
1. Set this token aside for later (we'll assume you wrote it to a file on your
   workstation at `/path/to/oauth/secret`)

## Tackle deployment

Prow's `tackle` utility walks you through deploying a new instance of prow in a couple minutes, try it out!

You need a few things:

1. [`bazel`](https://bazel.build/) build tool installed and working
1. The prow `tackle` utility. It is recommended to use it by running `bazel run //prow/cmd/tackle` from `test-infra` directory, alternatively you can install it by running `go get -u k8s.io/test-infra/prow/cmd/tackle` (in that case you would also need go installed and working).
1. Optionally, credentials to a Kubernetes cluster (otherwise, `tackle` will help you create one)

To install prow run the following from the `test-infra` directory and follow the on-screen instructions:

```bash
# Ideally use https://bazel.build, alternatively try:
#   go get -u k8s.io/test-infra/prow/cmd/tackle && tackle
bazel run //prow/cmd/tackle
```

The will help you through the following steps:

* Choosing a kubectl context (and creating a cluster / getting its credentials if necessary)
* Deploying prow into that cluster
* Configuring GitHub to send prow webhooks for your repos. This is where you'll provide the absolute `/path/to/oauth/secret`

See the [Next Steps](#next-steps) section after running this utility.

## Manual deployment

If you do not want to use the `tackle` utility above, here are the manual set of commands tackle will run.

Prow runs in a kubernetes cluster, so first figure out which cluster you want to deploy prow into. If you already have a cluster, skip to the next step.

You can use the [GCP cloud console](https://console.cloud.google.com/) to set up a project and [create a new Kubernetes Engine cluster](https://console.cloud.google.com/kubernetes).

### Create the cluster

I'm assuming that `PROJECT` and `ZONE` environment variables are set.

```sh
export PROJECT=your-project
export ZONE=us-west1-a
```

Run the following to create the cluster. This will also set up `kubectl` to
point to the new cluster.

```sh
gcloud container --project "${PROJECT}" clusters create prow \
  --zone "${ZONE}" --machine-type n1-standard-4 --num-nodes 2
```

### Create cluster role bindings

As of 1.8 Kubernetes uses [Role-Based Access Control (“RBAC”)](https://kubernetes.io/docs/admin/authorization/rbac/) to drive authorization decisions, allowing `cluster-admin` to dynamically configure policies.
To create cluster resources you need to grant a user `cluster-admin` role in all namespaces for the cluster.

For Prow on GCP, you can use the following command.

```sh
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin --user $(gcloud config get-value account)
```

For Prow on other platforms, the following command will likely work.

```sh
kubectl create clusterrolebinding cluster-admin-binding-"${USER}" \
  --clusterrole=cluster-admin --user="${USER}"
```

On some platforms the `USER` variable may not map correctly to the user
in-cluster. If you see an error of the following form, this is likely the case.

```console
Error from server (Forbidden): error when creating
"prow/cluster/starter.yaml": roles.rbac.authorization.k8s.io "<account>" is
forbidden: attempt to grant extra privileges:
[PolicyRule{Resources:["pods/log"], APIGroups:[""], Verbs:["get"]}
PolicyRule{Resources:["prowjobs"], APIGroups:["prow.k8s.io"], Verbs:["get"]}
APIGroups:["prow.k8s.io"], Verbs:["list"]}] user=&{<CLUSTER_USER>
[system:authenticated] map[]}...
```

Run the previous command substituting `USER` with `CLUSTER_USER` from the error
message above to solve this issue.

```sh
kubectl create clusterrolebinding cluster-admin-binding-"<CLUSTER_USER>" \
  --clusterrole=cluster-admin --user="<CLUSTER_USER>"
```

There are [relevant docs on Kubernetes Authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#authentication-strategies) that may help if neither of the above work.

### Create the GitHub secrets

You will need two secrets to talk to GitHub. The `hmac-token` is the token that
you give to GitHub for validating webhooks. Generate it using any reasonable
randomness-generator, eg `openssl rand -hex 20`.

```sh
# openssl rand -hex 20 > /path/to/hook/secret
kubectl create secret generic hmac-token --from-file=hmac=/path/to/hook/secret
```

The `oauth-token` is the OAuth2 token you created above for the [GitHub bot account]

```sh
# https://github.com/settings/tokens
kubectl create secret generic oauth-token --from-file=oauth=/path/to/oauth/secret
```

### Add the prow components to the cluster

Run the following command to deploy a basic set of prow components.

```sh
kubectl apply -f prow/cluster/starter.yaml
```

After a moment, the cluster components will be running.

```console
$ kubectl get deployments
NAME         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deck         2         2         2            2           1m
hook         2         2         2            2           1m
horologium   1         1         1            1           1m
plank        1         1         1            1           1m
sinker       1         1         1            1           1m
tide         1         1         1            1           1m
```

#### Get ingress IP address

Find out your external address. It might take a couple minutes for the IP to
show up.

```console
$ kubectl get ingress ing
NAME      HOSTS     ADDRESS          PORTS     AGE
ing       *         an.ip.addr.ess   80        3m
```

Go to that address in a web browser and verify that the "echo-test" job has a
green check-mark next to it. At this point you have a prow cluster that is ready
to start receiving GitHub events!

## Add the webhook to GitHub

Configure github to send your prow instance `application/json` webhooks
for specific repos and/or whole orgs.

You can do this with the `add-hook` utility:

```sh
# Note /path/to/hook/secret and /path/to/oauth/secret from earlier secrets step
# Note the an.ip.addr.ess from previous ingres step

# Ideally use https://bazel.build, alternatively try:
#   go get -u k8s.io/test-infra/experiment/add-hook && add-hook
bazel run //experiment/add-hook -- \
  --hmac-path=/path/to/hook/secret \
  --github-token-path=/path/to/oauth/secret \
  --hook-url http://an.ip.addr.ess/hook \
  --repo my-org/my-repo \
  --repo my-whole-org \
  --confirm=false  # Remove =false to actually add hook
```

Now go to your org or repo and click `Settings -> Webhooks`.

Look for the `http://an.ip.addr.ess/hook` you added above.
A green check mark (for a ping event, if you click edit and view the details of the event) suggests everything is working!

You can click `Add webhook` on the Webhooks page to add the hook manually
if you do not want to use the `add-hook` utility.

## Next Steps

You now have a working Prow cluster (Woohoo!), but it isn't doing anything interesting yet.
This section will help you configure your first plugin and job, and complete any additional setup that your instance may need.

### Enable some plugins by modifying `plugins.yaml`

Create a file called `plugins.yaml` and add the following to it:

```yaml
plugins:
  YOUR_ORG/YOUR_REPO:
  - size
```

Replace `YOUR_ORG/YOUR_REPO:` with the appropriate values. If you want, you can
instead just say `YOUR_ORG:` and the plugin will run for every repo in the org.

Next, create an empty file called `config.yaml`:

```sh
touch config.yaml
```

Run the following to test the files, replacing the paths as necessary:

```sh
bazel run //prow/cmd/checkconfig -- --plugin-config=path/to/plugins.yaml --config-path=path/to/config.yaml
```

There should be no errors. You can run this as a part of your presubmit testing
so that any errors are caught before you try to update.

Now run the following to update the configmap, replacing the path as necessary:

```sh
kubectl create configmap plugins \
  --from-file=plugins.yaml=path/to/plugins.yaml --dry-run -o yaml \
  | kubectl replace configmap plugins -f -
```

We added a make rule to do this for us:

```Make
get-cluster-credentials:
    gcloud container clusters get-credentials "$(CLUSTER)" --project="$(PROJECT)" --zone="$(ZONE)"

update-plugins: get-cluster-credentials
    kubectl create configmap plugins --from-file=plugins.yaml=plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -f -
```

Now when you open a PR, it will automatically be labelled with a `size/*`
label. When you make a change to the plugin config and push it with `make
update-plugins`, you do not need to redeploy any of your cluster components.
They will pick up the change within a few minutes.

### Set namespaces for prowjobs and test pods

Add the following to `config.yaml`:

```yaml
prowjob_namespace: default
pod_namespace: test-pods
```

By doing so, we keep prowjobs in the `default` namespace and test pods in the
`test_pods` namespace.

You can also choose other names. Remember to update the RBAC roles and
rolebindings afterwards.

**Note**: If you set or update the `prowjob_namespace` or `pod_namespace`
fields after deploying the prow components, you will need to redeploy them
so that they pick up the change.

### Configure Cloud Storage

When configuring Prow jobs to use the [Pod utilities](./pod-utilities.md)
with `decorate: true`, job metdata, logs, and artifacts will be uploaded
to a GCS bucket in order to persist results from tests and allow for the
job overview page to load those results at a later point. In order to run
these jobs, it is required to set up a GCS bucket for job outputs. If your
Prow deployment is targeted at an open source community, it is strongly
suggested to make this bucket world-readable.

In order to configure the bucket, follow the following steps:

1. [provision](https://cloud.google.com/iam/docs/creating-managing-service-accounts) a new service account for interaction with the bucket
1. [create](https://cloud.google.com/storage/docs/creating-buckets) the bucket
1. (optionally) [expose](https://cloud.google.com/storage/docs/access-control/making-data-public) the bucket contents to the world
1. [grant access](https://cloud.google.com/storage/docs/access-control/using-iam-permissions) to admin the bucket for the service account
1. [serialize](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) a key for the service account
1. upload the key to a `Secret` under the `service-account.json` key
1. edit the `plank` configuration for `default_decoration_config.gcs_credentials_secret` to point to the `Secret` above

After [downloading](https://cloud.google.com/sdk/gcloud/) the `gcloud` tool and authenticating,
the following script will execute the above steps for you:

```sh
gcloud iam service-accounts create prow-gcs-publisher # step 1
identifier="$(  gcloud iam service-accounts list --filter 'name:prow-gcs-publisher' --format 'value(email)' )"
gsutil mb gs://prow-artifacts/ # step 2
gsutil iam ch allUsers:objectViewer gs://prow-artifacts # step 3
gsutil iam ch "serviceAccount:${identifier}:objectAdmin" gs://prow-artifacts # step 4
gcloud iam service-accounts keys create --iam-account "${identifier}" service-account.json # step 5
kubectl -n test-pods create secret generic gcs-credentials --from-file=service-account.json # step 6
```

Before we can update plank's `default_decoration_config` we'll need to know the version we're using
```sh
$ kubectl get pod -lapp=plank -o jsonpath='{.items[0].spec.containers[0].image}' | cut -d: -f2
v20190619-25afbb545
```

Then, setup plank's `default_decoration_config` in `config.yaml`:
```yaml
plank:
  default_decoration_config:
    utility_images: # using the tag we identified above
      clonerefs: "gcr.io/k8s-prow/clonerefs:v20190619-25afbb545"
      initupload: "gcr.io/k8s-prow/initupload:v20190619-25afbb545"
      entrypoint: "gcr.io/k8s-prow/entrypoint:v20190619-25afbb545"
      sidecar: "gcr.io/k8s-prow/sidecar:v20190619-25afbb545"
    gcs_configuration:
      bucket: prow-artifacts # the bucket we just made
      path_strategy: explicit
    gcs_credentials_secret: gcs-credentials # the secret we just made
```

### Add more jobs by modifying `config.yaml`

Add the following to `config.yaml`:

```yaml
periodics:
- interval: 10m
  name: echo-test
  decorate: true
  spec:
    containers:
    - image: alpine
      command: ["/bin/date"]
postsubmits:
  YOUR_ORG/YOUR_REPO:
  - name: test-postsubmit
    decorate: true
    spec:
      containers:
      - image: alpine
        command: ["/bin/printenv"]
presubmits:
  YOUR_ORG/YOUR_REPO:
  - name: test-presubmit
    decorate: true
    always_run: true
    skip_report: true
    spec:
      containers:
      - image: alpine
        command: ["/bin/printenv"]
```

Again, run the following to test the files, replacing the paths as necessary:

```sh
bazel run //prow/cmd/checkconfig -- --plugin-config=path/to/plugins.yaml --config-path=path/to/config.yaml
```

Now run the following to update the configmap.

```sh
kubectl create configmap config \
  --from-file=config.yaml=path/to/config.yaml --dry-run -o yaml | kubectl replace configmap config -f -
```

We use a make rule:

```Make
update-config: get-cluster-credentials
    kubectl create configmap config --from-file=config.yaml=config.yaml --dry-run -o yaml | kubectl replace configmap config -f -
```

Presubmits and postsubmits are triggered by the `trigger` plugin. Be sure to
enable that plugin by adding it to the list you created in the last section.

Now when you open a PR it will automatically run the presubmit that you added
to this file. You can see it on your prow dashboard. Once you are happy that it
is stable, switch `skip_report` to `false`. Then, it will post a status on the
PR. When you make a change to the config and push it with `make update-config`,
you do not need to redeploy any of your cluster components. They will pick up
the change within a few minutes.

When you push or merge a new change to the git repo, the postsubmit job will run.

For more information on the job environment, see [`jobs.md`](/prow/jobs.md)

### Run test pods in different clusters

You may choose to run test pods in a separate cluster entirely. This is a good practice to keep testing isolated from Prow's service components and secrets. It can also be used to furcate job execution to different clusters.
Create a secret containing a `{"cluster-name": {cluster-details}}` map like this:

```yaml
default:
  endpoint: https://<master-ip>
  clientCertificate: <base64-encoded cert>
  clientKey: <base64-encoded key>
  clusterCaCertificate: <base64-encoded cert>
other:
  endpoint: https://<master-ip>
  clientCertificate: <base64-encoded cert>
  clientKey: <base64-encoded key>
  clusterCaCertificate: <base64-encoded cert>
```

Use [mkbuild-cluster][5] to determine these values:

```sh
bazel run //prow/cmd/mkbuild-cluster -- \
  --project=P --zone=Z --cluster=C \
  --alias=A \
  --print-entry | tee cluster.yaml
kubectl create secret generic build-cluster --from-file=cluster.yaml
```

Mount this secret into the prow components that need it (at minimum: `plank`,
`sinker` and `deck`) and set the `--build-cluster` flag to the location you mount it at. For
instance, you will need to merge the following into the plank deployment:

```yaml
spec:
  containers:
  - name: plank
    args:
    - --build-cluster=/etc/foo/cluster.yaml # basename matches --from-file key
    volumeMounts:
    - mountPath: /etc/foo
      name: cluster
      readOnly: true
  volumes:
  - name: cluster
    secret:
      defaultMode: 420
      secretName: build-cluster # example above contains a cluster.yaml key
```

Configure jobs to use the non-default cluster with the `cluster:` field.
The above example `cluster.yaml` defines two clusters: `default` and `other` to schedule jobs, which we can use as follows:

```yaml
periodics:
- name: cluster-unspecified
  # cluster:
  interval: 10m
  decorate: true
  spec:
    containers:
    - image: alpine
      command: ["/bin/date"]
- name: cluster-default
  cluster: default
  interval: 10m
  decorate: true
  spec:
    containers:
    - image: alpine
      command: ["/bin/date"]
- name: cluster-other
  cluster: other
  interval: 10m
  decorate: true
  spec:
    containers:
    - image: alpine
      command: ["/bin/date"]
```

This results in:

* The `cluster-unspecified` and `default-cluster` jobs run in the `default` cluster.
* The `cluster-other` job runs in the `other` cluster.

See [mkbuild-cluster][5] for more details about how to create/update `cluster.yaml`.

### Enable merge automation using Tide

PRs satisfying a set of predefined criteria can be configured to be
automatically merged by [Tide][6].

Tide can be enabled by modifying `config.yaml`.
See [how to configure tide][7] for more details.

#### Setup PR status dashboard

To setup a PR status dashboard like [prow.k8s.io/pr](https://prow.k8s.io/pr), follow the
instructions in [`pr_status_setup.md`](https://github.com/kubernetes/test-infra/blob/master/prow/docs/pr_status_setup.md).

### Configure SSL

Use [cert-manager][3] for automatic LetsEncrypt integration. If you
already have a cert then follow the [official docs][4] to set up HTTPS
termination. Promote your ingress IP to static IP. On GKE, run:

```sh
gcloud compute addresses create [ADDRESS_NAME] --addresses [IP_ADDRESS] --region [REGION]
```

Point the DNS record for your domain to point at that ingress IP. The convention
for naming is `prow.org.io`, but of course that's not a requirement.

Then, install cert-manager as described in its readme. You don't need to run it in
a separate namespace.

## Further reading

* [Developing for Prow](/prow/getting_started_develop.md)
* [Getting more out of Prow](/prow/more_prow.md)

[1]: https://github.com/settings/tokens
[2]: /prow/jobs.md#How-to-configure-new-jobs
[3]: https://github.com/jetstack/cert-manager
[4]: https://kubernetes.io/docs/concepts/services-networking/ingress/#tls
[5]: /prow/cmd/mkbuild-cluster/
[6]: /prow/cmd/tide/README.md
[7]: /prow/cmd/tide/config.md
[8]: https://github.com/kubernetes/test-infra/blob/master/prow/scaling.md#working-around-githubs-limited-acls
