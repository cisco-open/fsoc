## Adding Domain-Specific Configuration Parameters

We have two examples: UQL and knowledge.
They showcase the two different approaches fsoc's config extensions support:

1. Allow only one of a predetermined, enumerated set of versions you want to allow (e.g, just "v1" and "v2beta"). I think that's what we do in the UQL case
2. Allow any api-version--like value, e.g., v1, v1beta, v2alpha2,--even if it is not supported by the backend API. This is what we do in the knowledge case



## Testing domain-specific configuration

First, the domain-specific fields should show up in the output of the `fsoc config show-fields` command, just like the `uql.apiver` and `knowledge.apiver` fields below:

```
$ fsoc config show-fields
The following settings can be configured with the "config create" and "config set" commands.
The current setting values can be seen with the "config get" command.

Settings:
  auth             : authentication method, required. Must be one of "none", "oauth", "service-principal",
                     "agent-principal", "jwt", "local".
  url              : URL to the tenant, scheme and host/port only; required. For example,
                     https://mytenant.observe.appdynamics.com
  tenant           : tenant ID that is required only for auth methods that cannot automatically obtain it. Not needed
                     for the "oauth", "service-principal" and "local" auth methods.
  secret-file      : file containing login credentials for "service-principal" and "agent-principal" auth methods. The
                     file must remain available, as fsoc saves only the file's path.
  envtype          : platform environment type, optional. Used only for special development/test environments. If
                     specified, can be "dev" or "prod".
  token            : authentication token needed only for the "token" auth method.
  appd-tid         : value of appd-pid to use with the "local" auth method.
  appd-pty         : value of appd-pid to use with the "local" auth method.
  appd-pid         : value of appd-pid to use with the "local" auth method.
  server           : synonym for the "url" setting. Deprecated.
  knowledge.apiver : API version to use for knowledge store commands. The default is "v1".
  uql.apiver       : API version to use for UQL queries. The default is "v1".
```

To verify that the field can be set correctly to supported value (and fails, as expected for invalid values), you can try the following commands (replacing your newly added domain-specific parameter in the place of `uql.version`):

```
# Set the parameter to a valid value
$ fsoc config set uql.apiver=v2beta
Context "default" updated

# Verify that the parameter is recorded in the profile
$ fsoc config get
            Name: default
     Auth Method: oauth
     ...
      Subsystems:
                └ uql: apiver=v2beta

# Verify that the parameter has taken effect
$ fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)" --curl
...
   • curl command equivalent   command=curl -X 'POST' -d '{"query":"FETCH id, type, attributes FROM entities(k8s:workload)"}' -H 'Accept: application/json' -H 'Authorization: Bearer REDACTED' -H 'Content-Type: application/json' 'https://REDACTED.observe.appdynamics.com/monitoring/v2alpha/query/execute'
...

# Verify that invalid values are rejected
$ fsoc config set uql.apiver=foo
   ⨯ Failed to set subsystem-specific settings: failed to parse configuration for subsystem "uql": error decoding 'apiver': API version "foo" is not supported; valid value(s): "v1","v2beta"
   
# Clear setting (restore to default)
$ fsoc config set uql.apiver=""
Context "default" updated

# Verify setting is no longer present in the profile
$ fsoc config get
            Name: default
     Auth Method: oauth
     ...
     

```

