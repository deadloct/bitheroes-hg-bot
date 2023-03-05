# Discord Squid Game

> ⚠️ This bot is not affiliated with the Netflix hit Squid Game, but inspired by its plot.

This is a general-use contest bot. A sponsor begins the game by issuing the
`!squid-game [start-delay]` command, which opens up the competition. Contestants
enter by reacting to the bot's reply with a 🦑.

After `start-delay` seconds, the game commences through the Squid Game rounds, with
players being eliminated until one is eventually victorious.

## Adding New Deaths to the Game

The deaths are hardcoded in the `data/data.en.json` file, but they
will be abstracted eventually to a place where administrators can customize messages
for their specific purpose or server theme.

Deaths in `data/data.en.json` can use the following tokens:

* `{{DYING}}`: The player or team (such as in tug of war) that is dying.
* `{{LIVING}}`: A random player or team that has not died yet.

The `.en` prefix exists in the filename for future internationalization support.
