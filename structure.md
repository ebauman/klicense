# klicense project structure

```
klicense/
    api/
    cert/
    cli/
        klicense/
    client/
    codegen/
    example/
    hack/
    kubernetes/
    license/
    operator/
    remove/
```

### `/api`

This is where the Kubernetes types live, namely `Entitlement` and `Request`. 
This directory is both read from and written to during code generation. 
`zz_` files are written for things like deepcopy, registration, etc.

### `/cert`

This package contains the code for handling certs. 
Specifically this is where the code lives to generate RSA keys, read them in, and decode them.
Mostly used by the CLI.

### `/cli/klicense`

This is where the CLI lives. It is contained within a `/klicense` subdir so the output binary 
gets named nicely - that's all. 
Within this directory is the `main.go` for the CLI as well as the `cmd/` and related subdirs for 
subcommands when executing `klicense`.

### `/client`

This is the client code. In here is both the controller that watches/handles client-side actions
on `Request` objects, as well as the code necessary for the client to build those requests.
*This is what client softwares will import and use to interface with klicense.*

### `/codegen`

In this directory is the code generation components for the wrangler controllers present both in the 
client and in the operator. There are two subdirs, `/client` and `/operator` that achieve this.

### `/example`

In this directory lives two example applications to showcase blocking client and async client
functionality. 

### `/hack`

Standard Kubernetes `/hack` directory containing the boilerplate for codegen.

### `/kubernetes`

This is a struct and accompanying conversion method called `NamespacedName`.
This was created to allow for serialization of name+namespace of an object in k8s, such as a 
`Secret` or `Entitlement`. This is used in lieu of NamespacedName from `k8s.io/apimachinery/pkg/types`
because that type _doesn't have JSON tags so you can't serialize it_. So this was written to 
allow for JSON serialization of name and namespace, with a converter to get out
the apimachinery type when needed. 

### `/license`

This contains all the license generation code, including the embed of certificates for the 
operator and the client. **When using this library, change `embed.go` to include your own certificates!**
This package also contains the logic to validate a license from a Kubernetes secret, as well as some
helper methods for the CLI.

### `/operator`

The operator of klicense. Contains the controllers and logic to operate klicense in a Kubernetes cluster.
The controllers live in `/controllers`, `/crd` is the logic for installing CRDs into the cluster,
and `/generated` is where the wrangler codegenn'ed stuff lives.

### `/remove`

Used for registering remove handlers (finalizers) on Kubernetes objects that are maintained by a controller.
Shamelessly stolen from another project (thanks Arvind!).
It not only helps registering those handlers but also provides for filtering which objects
sholud be finalizer'ed.