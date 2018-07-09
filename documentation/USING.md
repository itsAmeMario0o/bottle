# Using bottle with Tetration

This guide covers how to use bottle to setup a 3 tier application that can be used within Tetration

## Setup 

### Software

You do not need much to run bottle, any releatively recent installation of Kubernetes, Helm, and Docker will be sufficient.

In this example the software versions are:

* `Kubernetes 1.11.0` 
* `Helm 2.9.1` 
* `Docker 1.13.1`

### Scale

Bottle scales well with minimal resources. You do not need a heavily provisioned kube environment to get started.

If you are new to docker and kubernetes, you may want to try the kube support in docker and run the examples on your laptop.

On a 2017 MacBook Pro 15" (16GB RAM) bottle could scale to around 60 pods.

In this example, a larger 8 node cluster will be used to demonstrate scale, comfortably running over 1000 pods.

## Guide

### Deploying Sensors

Download the CentOS 7.4 enforcer sensor from the target Tetration cluster

Generate an API key with the following credentials:



Create the docker image

```bash
> git clone https://cto-github.cisco.comcom/tigarner/bottle

> cd bottle

> cp ~/Downloads/tet-sensor-2.3.1.45-1.el7-pliny.enforcer.x86_64.rpm sensor/sensor.rpm

> cp ~/Downloads/api_credentials.json sensor/

> docker build -f sensor/Dockerfile -t bottle:pliny .
```

Tag the image and push to the docker registry

```bash
docker tag bottle:pliny tigarner/bottle:pliny
```

Deploy the "3tier" scenario

```yaml
# scenarios/3tier.yaml
scenario: 3tier
ships:
    web: 
        replicas: 3
        clients:
        - app:80
    app: 
        replicas: 3
        servers:
        - 80
        clients:
        - db:3306
    db:
        replicas: 3
        servers:
        - 3306
```

```
helm install -f scenarios/3tier.yaml --set image=tigarner/bottle:pliny --set scope=Bottle bottle
NAME:   pondering-octopus
LAST DEPLOYED: Fri Jul  6 22:53:18 2018
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1/ConfigMap
NAME          DATA  AGE
3tier-config  3     1s

==> v1/Service
NAME  TYPE       CLUSTER-IP  EXTERNAL-IP  PORT(S)   AGE
db    ClusterIP  None        <none>       3306/TCP  1s
web   ClusterIP  None        <none>       <none>    1s
app   ClusterIP  None        <none>       80/TCP    1s

==> v1beta1/Deployment
NAME  DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
db    3        3        3           0          1s
web   3        3        3           0          1s
app   3        3        3           0          1s

==> v1/Pod(related)
NAME                  READY  STATUS             RESTARTS  AGE
db-5d4f85bc7c-65lcr   0/3    ContainerCreating  0         1s
db-5d4f85bc7c-7x5nl   0/3    ContainerCreating  0         1s
db-5d4f85bc7c-82fpf   0/3    ContainerCreating  0         1s
web-79b7fff49f-fp7d9  0/3    ContainerCreating  0         1s
web-79b7fff49f-ks2rc  0/3    ContainerCreating  0         1s
web-79b7fff49f-zp2jm  0/3    ContainerCreating  0         1s
app-58cd74b8c8-92qvn  0/3    ContainerCreating  0         1s
app-58cd74b8c8-qnwtn  0/3    ContainerCreating  0         1s
app-58cd74b8c8-wm2js  0/3    ContainerCreating  0         1s


NOTES:
Installed bottle scenario 3tier

Your release is named pondering-octopus.

To learn more about the release, try:

  $ helm status pondering-octopus
  $ helm get pondering-octopus

To delete the release:

  $ helm delete pondering-octopus
```

In the above output you can observe a number of resources have been created

Each ship (web, app, db) will be created as a deployment with three replicas as defined in the scenario
```
==> v1beta1/Deployment
NAME  DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
db    3        3        3           0          1s
web   3        3        3           0          1s
app   3        3        3           0          1s
```

Each replica will run a seperator traffic generator and sensor container
```
==> v1/Pod(related)
NAME                  READY  STATUS             RESTARTS  AGE
db-5d4f85bc7c-65lcr   0/3    ContainerCreating  0         1s
db-5d4f85bc7c-7x5nl   0/3    ContainerCreating  0         1s
db-5d4f85bc7c-82fpf   0/3    ContainerCreating  0         1s
web-79b7fff49f-fp7d9  0/3    ContainerCreating  0         1s
web-79b7fff49f-ks2rc  0/3    ContainerCreating  0         1s
web-79b7fff49f-zp2jm  0/3    ContainerCreating  0         1s
app-58cd74b8c8-92qvn  0/3    ContainerCreating  0         1s
app-58cd74b8c8-qnwtn  0/3    ContainerCreating  0         1s
app-58cd74b8c8-wm2js  0/3    ContainerCreating  0         1s
```


The services will be provided by the named ships
```
==> v1/Service
NAME  TYPE       CLUSTER-IP  EXTERNAL-IP  PORT(S)   AGE
db    ClusterIP  None        <none>       3306/TCP  1s
web   ClusterIP  None        <none>       <none>    1s
app   ClusterIP  None        <none>       80/TCP    1s
```


### Validating Sensors

Each pod will run a unique sensor, which will register with Tetration as an enforcement sensor

The reported IPs will be local to the Kube cluster and may be different depending on your setup, in this example they are in 192.168.0.0/16

If the sensors do not show up, you may need to check the status of the pods using `kubectl get pods -l scenario=3tier` or `helm status pondering-octopus` to check the state of pods.

If all pods are running and the sensors do not appear, you may need to use Remote VRF configuration to assign sensors to the right Tenant. 

Assuming no other NAT, you can use the public IPs of the kube nodes as the source IP for the Remote VRF configuration

![Remote VRF configuration](vrfs.png)

All 9 pods are successfully connected and sending telemetry

![Connected sensors](agent-list.png)

After a few minutes each agent will have a host profile page, where you can see four annotations have been created

![Host Profile](host-profile.png)

You can also view the status of the bottle resources on the Kubernetes dashboard

![Dashboard Overview](dashboard-overview.png)

### Annotations and Scopes

Each bottle host will get four annotions, we can search the inventory for `bottle_lifecycle=active and OS=CentOS` workloads:

![Inventory](inventory.png)

>Take note: when a pod terminates, it will attempt to annotate itself with bottle_lifecycle=terminated, however if for any reason
>a pod is unexpectedly terminated it may fail to update the annotation, luckily after some time the agent will be unregistered and therefore the
>OS field will become blank

You can now quite easily create a scope tree to classify bottle endpoints using Annotations

![Scope 1](scope1.png)
![Scope 1](scope2.png)

### Application Workspace

Tetration will now be collecting data and assigning it the created scope tree continously. 

If you want to run ADM, you will need to leave the system running for 6 hours or more now.

If you want to write policy by hand, you can use the annotations to create dynamic clusters right now - jump to [Dynamic Policy](#dynamic-policy) for some tips.

Create the application workspace for the 3tier scenario, and make sure to select Dynamic.

![Workspace](workspace.png)

To run ADM, most settings should be ok set to defaults, the only change of note is I created two filters restricted to the "Bottle" scope to represent the internal kubernetes DNS and Tetration IPs.

>![Filters](filters.png)

![Settings](settings.png)

If everything goes as expected, you should see ADM results that are exactly in line with the Scenario specification


![Policy 1](policy1.png)
![Policy 2](policy2.png)

### Analysing Policy

Now that the application policy has been discovered, we can test the policy against the constantly generated traffic to ensure we won't break anything

### Enforcing Policy

All bottle pods can have policy enforced in them! Yes, even though it is a container there is a full local instance of iptables

>Note: bottle uses full (e.g. for VMs) enforcement sensors INSIDE the container image. This is NOT the recommended way to enforce policy inside 
>containers using Tetration. If you are using Tetration to enforce policy in normal application containers please see the container 
>enforcement documentation to understand how to install and use the feature.

After clicking enforce in the Tetration UI, after a minute or so, you can get the iptable output from any of the running pods:

### Dynamic Policy

What happens if we scale the scenario to 5, 10, 100, 1000 replicas? Our policy intent stays the same, but the workloads it applies to must dymaically scale
to correctly classify and apply policy to the new endpoints.

In Tetration we can convert the regular clusters into dynamic clusters based on the bottle annotations:

We can then easily scale up the scenario replicas, how about 10:

Or what about 300!

Tetration seamlesslessly onboards the new agents, correctly assigns them into the clusters based on annotations, and finally applies the `iptable` security rules, all without any further administrator interaction.

And finally we can scale the application back down to our original three replicas:

And policy contracts as necessary

### Cleaning Up

We could now leave the scenario running indefinitely and use this as a stable example setup, or we could choose to tear down the environment and leave no trace.

To remove the scenario, execute the helm command to delete the release name:

`helm delete pondering-octopus`