# GizClaw Service Tree

This document describes business-level services, not transport service IDs.

Peer Service and Device Service are peer-provided services. Public Service,
Gear Service, and Admin Service are server-provided services.

## Doc Style

- Business RPC-style methods use dotted names: `service.resource.method`.
- Multiple methods on one business resource use braces: `resource.{list,get,create,put,delete}`.
- Resource endpoints use path-first notation: `/path OPERATION[, OPERATION]`.
- Custom HTTP verbs are listed under their resource path as subtree items: `@verb`.

```text
Peer Service
└── peer.ping

Device Service
├── device.info.get
└── device.identifiers.get

Public Service
├── server.info.get
├── /server-info GET
└── /login POST

Gear Service
├── peer.info.{get,put}
├── peer.runtime.get
├── peer.status.{get,put}
├── audio.say
├── workspace.{list,get,create,put,delete}
├── workflow.{list,get,create,put,delete}
├── model.{list,get,create,put,delete}
├── credential.{list,get,create,put,delete}
├── peer.run.agent.{get,set}
├── peer.run.{reload,status,stop}
├── pet.{list,get,create,put,delete}
├── pet.feed
├── pet.play
├── pet.level-up
├── wallet.get
├── wallet.transactions.list
├── contact.{list,get,create,put,delete}
├── contact.block
├── contact.unblock
├── friend.requests.{list,create}
├── friend.requests.accept
├── friend.requests.reject
├── friend.{list,delete}
├── group.{list,get,create,put,delete}
├── group.members.{list,add,delete}
├── group.messages.{list,send}
├── call.{list,get,create}
├── call.answer
├── call.reject
├── call.end
├── game.results.create
├── reward.{list,get,create}
└── reward.claim

Admin Service
├── /@apply POST
├── /resources/{kind}/{name} GET, PUT, DELETE
├── /acl/views/{name} LIST, CREATE, GET, PUT, DELETE
├── /acl/roles/{name} LIST, CREATE, GET, PUT, DELETE
├── /acl/policy-bindings/{id} LIST, CREATE, GET, PUT, DELETE
├── /workflows/{name} LIST, CREATE, GET, PUT, DELETE
├── /firmwares/{name} LIST, CREATE, GET, PUT, DELETE
│   ├── @release
│   └── @rollback
├── /credentials/{name} LIST, CREATE, GET, PUT, DELETE
├── /models/{id} LIST, CREATE, GET, PUT, DELETE
├── /dashscope-tenants/{name} LIST, CREATE, GET, PUT, DELETE
├── /gemini-tenants/{name} LIST, CREATE, GET, PUT, DELETE
├── /openai-tenants/{name} LIST, CREATE, GET, PUT, DELETE
├── /minimax-tenants/{name} LIST, CREATE, GET, PUT, DELETE
│   └── @sync-voices
├── /volc-tenants/{name} LIST, CREATE, GET, PUT, DELETE
│   └── @sync-voices
├── /voices/{id} LIST, CREATE, GET, PUT, DELETE
├── /workspaces/{name} LIST, CREATE, GET, PUT, DELETE
├── /peers/{publicKey}/pets/{id} LIST, GET
├── /peers/{publicKey}/wallet GET
│   └── /transactions LIST, GET
├── /peers/{publicKey}/contacts/{id} LIST, CREATE, GET, PUT, DELETE
│   ├── @block
│   └── @unblock
├── /peers/{publicKey}/friend-requests/{id} LIST, CREATE, GET, PUT, DELETE
│   ├── @accept
│   └── @reject
├── /peers/{publicKey}/friends/{id} LIST, GET, DELETE
├── /groups/{id} LIST, CREATE, GET, PUT, DELETE
│   ├── /members LIST, CREATE, GET, DELETE
│   └── /messages LIST, CREATE, GET
├── /calls/{id} LIST, CREATE, GET
│   ├── @answer
│   ├── @reject
│   └── @end
├── /game-results/{id} LIST, CREATE, GET
├── /rewards/{id} LIST, CREATE, GET
│   └── @claim
├── /peers/{publicKey} LIST, GET, DELETE
│   ├── /info GET, PUT
│   ├── /config GET, PUT
│   ├── /runtime GET
│   ├── /status GET
│   ├── @approve
│   ├── @block
│   └── @refresh
└── /peers
    ├── @findPubKeyBySn
    └── @findPubKeyByImei
```
