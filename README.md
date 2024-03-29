# Bit Heroes Hunger Games Bot Remake

This is a general contest bot based on Shadown's original Hunger Games bot used by the Bit Heroes community.

A sponsor begins the a game by issuing the `/hg-start` slash command, which starts a competition. Contestants enter by reacting to the bot with a specific ️emoji.

After `start-delay` seconds, the game commences with players being eliminated regularly until `victors` competitors remain.

## Running the Bot

This bot is written in go. See [go.dev](https://go.dev/) for installation instructions.

After cloning this repository, create a file in the root directory called `.env` and copy these lines into it:

```bash
export BITHEROES_HG_BOT_AUTH_TOKEN=
export BITHEROES_HG_BOT_EMOJI_NAME=
export BITHEROES_HG_BOT_EMOJI_ID=
```

You'll need to create an application, and then a bot under that application, on the Discord developer site. Enter the new bot's authorization token after the `BITHEROES_HG_BOT_AUTH_TOKEN=` line above.

Next you'll need to set the emoji name and ID equal to the reaction emoji that you'd like to use for the bot. 

* The emoji name is easy to find, just hover above the emoji after it's been sent to a channel and use the part between the colons. For example for `:hungergames:` use `hungergames`.
* To find the ID, right click on the emoji in a channel and select Copy Link. Use the webp file name without the extension as the ID. For example for the URL `https://cdn.discordapp.com/emojis/1084494508248543383.webp?size=96&quality=lossless` use `1084494508248543383`.

Afterward start the bot by running:

```bash
make run 
```

## Running Tests

While test coverage is slim at the moment, running tests is worthwhile after adding new phrases to test that they can be parsed properly.

Run tests using make:

```bash
make test
```

The benchmark tests require the benchstat program to compare old and new results:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Then run the tests:

```bash
make bench
```

## Adding New Phrases

The death phrases are imported from `data/phrases.en.json` file and can use the following tokens:

* `{{.Dying}}`: The player that is dying.
* `{{.Killer}}`: A random living player that contributed to the dying player's death.

## Credits

Thanks to Shadown for the original Bit Heroes Hunger Games bots. This bot is nothing but a cheap, unworthy imitation.

I need to also credit the phrase authors. It may take time to troll the galaxy for their messages, as they've done with their terrible and offensive humor.

Speaking of terrible and offensive humor, the jester's jokes were borrowed from [https://github.com/yesinteractive/dadjokes](https://github.com/yesinteractive/dadjokes).
