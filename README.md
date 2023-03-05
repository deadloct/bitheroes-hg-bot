# Bit Heroes Hunger Games Bot

This is a general-use contest bot. A sponsor begins the game by issuing the
`!hg [start-delay]` command, which opens up the competition. Contestants
enter by reacting to the bot with a 🕊️.

After `start-delay` seconds, the game commences through games with
players being eliminated until one is eventually victorious.

## Adding New Deaths to the Game

The deaths are hardcoded in the `data/settings.en.json` file and can use the following
tokens:

* `{{DYING}}`: The player or team (such as in tug of war) that is dying.
* `{{LIVING}}`: A random player or team that has not died yet.
