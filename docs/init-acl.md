# Init ACL Config

A user can configure the ACL of etcd by providing an **init-acl** config
in the config file, (See [config.example.yaml](../config.example.yaml) for examples).

The ACL config will be applied by the **Seeder** during provision, and it's **ONLY** applied once during that period.
After that, if a user wants to update the **init-acl** config, he needs to restart the **Seeder**,
or kill the **Seeder** and wait for another node to become the **Seeder**.

Once the **init-acl** is applied, the [etcd authentication](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/authentication.md) will be turned on.
The operator will not turn off the etcd authentication by itself, and after that moment,
only a user with "root" access to the etcd are able to turn it off manually with ```etcdctl auth disable```.

The **init-acl** config contains 3 parts, `rootPassword`, `roles` and `users`.

### rootPassword

The `rootPassword` is the password for the root user, it's optional.
An etcd client could provide the `rootPassword` (if it's not empty),
or provide a signed TLS ceritificate with `CN = root` (if the `rootPassword` is empty) to authenticate as a `root` user without password.

### Roles

The `roles` section defines a list of roles with their permissions.
The permissions are consist of a list of range keys, mode, whether the key is prefixed.

E.g.

```
- mode: readwrite
  key: /registry
  prefix: true
```
Allows the `readwrite` permission on all the paths whose prefix is `/registry`, such as `/registry/foo`, `/registry/bar`, etc.

```
- mode: read
  key: /foo1
  rangeEnd: /foo5
```
Allows the `read` permission on paths from `/foo1` to `/foo5`.


### Users

The `users` section defines a list of users, each user can be assigned to multiple roles.
Optionally, a password can be also set for the user.
Without a password, etcd will checks the client's TLS cert and use the `CommonName (CN)` to authenticate the user.
