## Docker Testing

The docker compose file within this directory allows one to quickly get a 3-nodes _etcd-cloud-operator_ cluster up and
running on their laptop, for testing or development purposes. Running it is extra simplistic:

```
docker-compose up
```

Discovery is performed by the internal _docker_ provider which simply looks up running containers and filters them by 
name. Snapshots are stored in the filesystem at `/var/lib/snapshots`. The first instance exposes all available ports
on the host, while the others only expose their ports to other instances. TLS is only present between peers, clients
communications are plain. Refer to the [config.yaml](config.yaml) for more details and customizations.

```
docker exec -it docker-testing_eco-0_1 sh -c 'IP=$(ip route get 1 | awk '"'"'{print $NF;exit}'"'"'); etcdctl --endpoints=$IP:2379 member list -w table'
+------------------+---------+------------------------+--------------------------+-------------------------+------------+
|        ID        | STATUS  |          NAME          |        PEER ADDRS        |      CLIENT ADDRS       | IS LEARNER |
+------------------+---------+------------------------+--------------------------+-------------------------+------------+
| 1219b3d6ddc417f6 | started | docker-testing_eco-0_1 | https://192.168.0.2:2380 | http://192.168.0.2:2379 |      false |
| 5ac283d796e472ba | started | docker-testing_eco-2_1 | https://192.168.0.4:2380 | http://192.168.0.4:2379 |      false |
| c65bf70f71ae37a6 | started | docker-testing_eco-1_1 | https://192.168.0.3:2380 | http://192.168.0.3:2379 |      false |
+------------------+---------+------------------------+--------------------------+-------------------------+------------+
```
