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


# Validating Sensors

![Connected sensors](agent-list.png)

