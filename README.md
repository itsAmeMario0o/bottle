# Bottle

>simple, flexible, and scalable application-like traffic simulator

Bottle uses Kubernetes to deploy distributed stateful traffic generators ("ships") that behave like real application components.

Bottle allows the user to spin up complex and high scaling scenarios with very little configuration or resources.

Bottle can be integrated with a Tetration sensor package to monitor and enforce segmentation policy.

## Getting Started

To use bottle, you will need:

* A kubernetes cluster
* A machine with `kubectl` installed and configured
* A machine with `docker` installed and configured
* A machine with `go` installed and configured

### Quick Start

```bash
# clone the repository
> git clone https://github.com/amney/bottle
> cd bottle

# install the bottle application
> go get -d .
> go install bottle

# copy your clusters agent rpm
> cp ~/Downloads/myclustersensor.rpm sensor/sensor.rpm

# copy your clusters api credentials (must have sw agent privilege)
> cp ~/Downloads/api_credentials.json sensor/api_credentials.rpm

# build the image and tag
> docker build --rm -f Dockerfile -t bottle:yourcluster .

# run a scenario with the image
> bottle -i bottle:yourcluster -f scenarios/3tier.yaml

# check the pods are running
> kubectl get pods -o wide -l scenario=3tier

# remove the created objects
> kubectl delete all -l scenario=3tier
```

## Overview

Bottle uses scenario templates to describe a set of application tiers and the communication patterns between.

An example bottle scenario to simulate a three tier web app, with traffic flowing between each tier

```c
[web]--[80]-->[app]--[3306]-->[db]
```

```yaml
  name: 3tier
  ships:
    web: |
        replicas: 3
        clients:
        - app:80

    app: |
        replicas: 2
        servers:
        - 80
        clients:
        - db:3306

    db: |
        replicas: 3
        servers:
        - 3306
```

`bottle` will parse this file and create the necessary kubernetes components (services & pods) to simulate the environment.

Load generators will run on each tier sending and receiving traffic as laid out in the spec.

## Scenarios

Scenarios describe the application components and the traffic between them, plus some optional metadata. 

A scenario file contains a name and a list of application components that will be mimicked by traffic generator containers, known as "ships".

Each `ship` starts with a key to describe the component name, and:

* `replicas` describes the count of endpoints in this component
* `clients` describes the outgoing connections from this component
* `servers` describes the listening sockets on this component

Each `client` must be a hostname and port number, like `db:3306`.

Each `server` must be a port number.

## Generator Image

The traffic generator pods will be deployed using an image you provide, this image must include a sensor and set of api credentials for the cluster you wish to analyse the traffic on.

### Sensor

The sensor you provide should meet the following critiria:

* Deep Visibility or Enforcement
* CentOS 7.4

### Sensor

The API credentials you provide should meet the following critiria:

* For the same cluster as the provided sensor
* Have the "SW sensor management: API to configure and monitor status of SW sensors" capability

### Tagging the Image

When building (or after) please tag the image and, if desired, push your image to a repository the target Kubernetes cluster has access to.

## Tips

### Remote VRF

If you wish to assign the agents created by scenarios to a custom tenant, utilize remote vrf configuration on the cluster to assign the public addresses of the nodes in the kubernetes.

### Inspecting

When you have deployed a scenario, you may be interested to check the status.

To view logs for the sensor for a given `<scenario> <component>` you must also choose sensor or traffic generator logs.

```bash
> kubectl logs -l scenario=<scenario> <component> [sensor|generator]
```

To attach into the shell of a component you must also choose sensor or generator shell:

```bash
> kubectl exec -it <component> -c [sensor|generator] -- /bin/bash
```

## Project Goals

This project is in a alpha stage and should be seen as a minimum viable implementation.

The first goal of being able to generate application like traffic that is useful for Tetration application dependency maps has been achieved.

The current goal of the project is to enrich the scenario specification DSL to include more parameters like configurable traffic pattern, injecting network latency, and application latency.






