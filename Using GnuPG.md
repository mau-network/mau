* GnuPG is a solid implementation for PGP. for Mau to have it as part of the specification will introduce some advantages and disadvantages along with it.

# Using GnuPG as identity backend for Mau


## Advantages
* Simpler to implement on systems that has `gpg` installed
* `gpg` has a machine readable output option
* GPG already implemented many features that Mau can simply use
* GpgME library can be used instead of calling `gpg` process to avoid requiring `gpg` binary to be installed
* GPG interfaces such as Kleopatra can be use to manage accounts and followed/friends keys
* User can reuse their already existing GPG keys and imported keys
* User can use `gpg` CLI to operate on their keys and imported friends public keys
* The `~/.gnupg` directory on *nix machines has a documented structure and the technical depth for future Mau expansions such as trust levels
* GPG can do maintainance operations such as generating revocation, sending keys, updating trustdb file.

## Disadvantages
* Implementer must learn GPG CLI if they're going to invoke the process from their application
* If the implementer doesn't want to invoke `gpg` in their process they will have to use GpgME C library or a binding for their language. which is not a simple library to use
* For platforms that doesn't have `gpg` installed the implementation will either
  * use GpgME or the system keyring services.
  * or use the system keyring/keystore service. which is different for Android and iOS. this will open the door for guessing the implementation between different applications trying to reuse the same account for different data types
  * Or compile GpgME library for the target platform. which is a huge effort for a developer
* Storing account information separate from the content data so backing up an account is a different process than backing up the content

# Decision
* To be decided

#decision