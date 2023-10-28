# audiusd

the simpliest way to run and interact with an audius node.

## install

```
curl -sSL https://raw.githubusercontent.com/AudiusProject/dot-slash-audius/main/install.sh | sh
```

## quickstart

run a dev setup

```
mkdir ~/.audius && cp sample.audius.conf ~/.audius/audius.conf

audius
```

## run

**configure**

minimal required config, (default location `~/.audius/audius.conf`) or pass via `-c` flag at runtime

```
# creator-node audius.conf
creatorNodeEndpoint=
delegateOwnerWallet=
delegatePrivateKey=
spOwnerWallet=
```

```
# discovery-provider audius.conf
audius_discprov_url=
audius_delegate_owner_wallet=
audius_delegate_private_key=
```

**run**
```
audius [-c audius.conf]
```

## build

builds required go binaries that are (for now) committed to this repo on the `bin` branch by CI.

```
make
```
