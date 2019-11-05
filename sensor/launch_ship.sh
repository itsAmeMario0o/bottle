#!/bin/bash

# install the Tetration sensor and get a uuid
rpm -Uvh --nodeps sensor.rpm
echo "ACTIVATION_KEY=$2" > /usr/local/tet/user.cfg
echo "HTTPS_PROXY=$3" >> /usr/local/tet/user.cfg
cd /usr/local/tet/ && ./fetch_sensor_id.sh

# start the Tetration sensor
cd /usr/local/tet && /usr/sbin/tet-engine tet-sensor -f conf/.sensor_config &
cd /usr/local/tet && /usr/sbin/tet-engine -n -c tet-main &
cd /usr/local/tet && /usr/sbin/tet-engine -n tet-enforcer --logtostderr &

# launch the traffic generator
cd / && mv ship $1; ./$1