# apod-bot

A [go](https://go.dev/) discord bot that fetches and posts [Astronomy Picture of the Day](https://apod.nasa.gov/apod/).

## Features

- Scheduled posting with `/schedule` and `/stop`
- Manually posting today's picture with `/today`
- All Astronomy Picture of the Day API calls are appropriately cached.

## Usage

**Currently, the bot is in development. Use at your own risk.**

<https://discord.com/api/oauth2/authorize?client_id=952282891910512663&permissions=18432&scope=applications.commands%20bot>

## Development

`APOD_TOKEN` token and `DISCORD_TOKEN` are passed in as environment variables. These may be set in a `.env` file like so:

```text
APOD_TOKEN=<token>
DISCORD_TOKEN=<token>
```

To learn more about discord bot development, visit [discord developers](https://discord.com/developers/docs/intro) docs](https://discord.com/developers/docs/intro). To create a NASA API token visit [api.nasa.gov](https://api.nasa.gov/index.html#authentication).
