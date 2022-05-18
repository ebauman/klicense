# klicense

klicense is a project for doing software licensing on kubernetes for go projects.

## Usage

You're going to want to fork this most likely.

Because we need to embed certificates in the operator and the client, you'll need to assign your own. 

Fork this repo, and put new public keys in `license/embed.go` (make sure you hold onto the private keys)

Then you can use `cli/klicense` to create licenses using the keys, and the operator to validate/assign.