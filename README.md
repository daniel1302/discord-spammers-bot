# discord-spammers-bot

The discord bot that register deleted message and send it to some dedicated channel. The bot also monitors new messages and check if there are monitored keywords.

## Required permissions

- `Message Content Intent`
- `bot.Send Messages`
- `bot.Read Message History`
- `bot.Manage Messages` - if you enable the `delete_invite_links` feature

## Example

```
Usage:

go build -o bot ./

./bot config.toml
```