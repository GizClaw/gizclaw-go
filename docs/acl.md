# GizClaw ACL Matrix

This file lists the target ACL-managed subjects, resources, permissions, and
business actions for Gear Service and Admin Service.

The current ACL schema already supports `workspace`, `workflow`, `model`,
`credential`, `voice`, and `view`. The other resource kinds below are planned
business extensions and must be added to the ACL schema before implementation.

## Target ACL-Controlled Resource Kinds

```text
workspace
workflow
model
credential
voice
view
pet
wallet
contact
friend
friend_request
group
call
game_result
reward
```

## Subjects

| Subject kind | ID | Meaning |
| --- | --- | --- |
| `pk` | gear public key | One gear or peer identity. |
| `view` | view name | A grouped subject for curated access. |
| `all_peers` | empty | Default subject that every connected peer can inherit. |

## Resource Matrix

| Resource kind | ID | Owner | Permissions | Gear Service usage | Admin Service usage |
| --- | --- | --- | --- | --- | --- |
| `workspace` | workspace name | gear or admin | `workspace.{read,use,admin}` | `workspace.{list,get,create,put,delete}`, `peer.run.reload` | `/workspaces/{name}` |
| `workflow` | workflow name | gear or admin | `workflow.{read,use,admin}` | `workflow.{list,get,create,put,delete}`, `peer.run.reload` | `/workflows/{name}` |
| `model` | model id | gear or admin | `model.{read,use,admin}` | `model.{list,get,create,put,delete}`, `peer.run.reload`, `audio.say` | `/models/{id}` |
| `credential` | credential name | gear or admin | `credential.{read,use,admin}` | `credential.{list,get,create,put,delete}`, `peer.run.reload`, `audio.say` | `/credentials/{name}` |
| `voice` | voice id | admin | `voice.{read,use,admin}` | `audio.say`, voice selection, and runtime use | `/voices/{id}` |
| `view` | view name | admin | `view.{read,use,admin}` | read/use resources exposed by a view | `/acl/views/{name}` |
| `pet` | pet id | gear | `pet.{read,use,admin}` | `pet.{list,get,create,put,delete}`, `pet.feed`, `pet.play`, `pet.level-up` | `/peers/{publicKey}/pets/{id}` |
| `wallet` | gear public key | gear | `wallet.{read,use,admin}` | `wallet.get`, `wallet.transactions.list` | `/peers/{publicKey}/wallet` |
| `contact` | contact id | gear | `contact.{read,use,admin}` | `contact.{list,get,create,put,delete}`, `contact.block`, `contact.unblock` | `/peers/{publicKey}/contacts/{id}` |
| `friend` | friend relation id | gear pair | `friend.{read,use,admin}` | `friend.{list,delete}` | `/peers/{publicKey}/friends/{id}` |
| `friend_request` | request id | gear pair | `friend_request.{read,use,admin}` | `friend.requests.{list,create}`, `friend.requests.accept`, `friend.requests.reject` | `/peers/{publicKey}/friend-requests/{id}` |
| `group` | group id | gear or admin | `group.{read,use,admin}` | `group.{list,get,create,put,delete}`, `group.members.{list,add,delete}` | `/groups/{id}` |
| `call` | call id | gear pair or group | `call.{read,use,admin}` | `call.{list,get,create}`, `call.answer`, `call.reject`, `call.end` | `/calls/{id}` |
| `game_result` | result id | gear | `game_result.{read,use,admin}` | `game.results.create` | `/game-results/{id}` |
| `reward` | reward id | gear or system | `reward.{read,use,admin}` | `reward.{list,get,create}`, `reward.claim` | `/rewards/{id}` |

## Permission Mapping

| Permission suffix | Meaning |
| --- | --- |
| `read` | List or get metadata/state for a resource. |
| `use` | Use the resource at runtime or perform normal owner actions. |
| `admin` | Create, update, delete, or manage ACL for the resource. |

## Runtime ACL Checks

| Operation | Required ACL checks |
| --- | --- |
| `peer.run.agent.set` | target `workspace.use` |
| `peer.run.reload` | current pending agent workspace `workspace.use`, `workflow.use`, referenced `model.use`, referenced `credential.use` |
| `audio.say` | selected `voice.use`, selected TTS `model.use`, referenced `credential.use` |
| `group.messages.list` | `group.read` |
| `group.messages.send` | `group.use` |
| reward settlement into wallet | `reward.use`, target `wallet.use` |

## Default Ownership Rules

| Create path | Subject to bind | Resource to bind | Role |
| --- | --- | --- | --- |
| Gear creates workspace | `pk:{gearPublicKey}` | `workspace:{name}` | workspace owner/admin |
| Gear creates workflow | `pk:{gearPublicKey}` | `workflow:{name}` | workflow owner/admin |
| Gear creates model | `pk:{gearPublicKey}` | `model:{id}` | model owner/admin |
| Gear creates credential | `pk:{gearPublicKey}` | `credential:{name}` | credential owner/admin |
| Gear creates pet | `pk:{gearPublicKey}` | `pet:{id}` | pet owner/admin |
| Gear starts wallet | `pk:{gearPublicKey}` | `wallet:{gearPublicKey}` | wallet owner/admin |
| Gear creates contact | `pk:{gearPublicKey}` | `contact:{id}` | contact owner/admin |
| Gear creates group | `pk:{gearPublicKey}` | `group:{id}` | group owner/admin |
| Gear creates game result | `pk:{gearPublicKey}` | `game_result:{id}` | game result owner/admin |
| Gear creates reward | `pk:{gearPublicKey}` | `reward:{id}` | reward owner/admin |
| System creates reward for gear | `pk:{gearPublicKey}` | `reward:{id}` | reward owner/user |

## Shared Resource Rules

| Shared resource | Subject | Resource | Role |
| --- | --- | --- | --- |
| Built-in model for everyone | `all_peers` | `model:{id}` | model reader/user |
| Built-in model for a group | `view:{name}` | `model:{id}` | model reader/user |
| Shared credential for one gear | `pk:{gearPublicKey}` | `credential:{name}` | credential user |
| Shared credential for a group | `view:{name}` | `credential:{name}` | credential user |
| Shared voice for everyone | `all_peers` | `voice:{id}` | voice reader/user |
