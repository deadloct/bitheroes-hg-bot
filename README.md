# Bit Heroes Hunger Games Bot

This is a general-use contest bot. A sponsor begins the game by issuing the
`!hg [start-delay]` command, which opens up the competition. Contestants
enter by reacting to the bot with a 🕊️.

After `start-delay` seconds, the game commences with players being eliminated regularly
until one remains.

## Adding New Phrases to the Game

The death phrases are imported from `data/phrases.en.json` file and can use the following
tokens:

* `{{.Dying}}`: The player that is dying.
* `{{.Killer}}`: A random living player that contributed to the dying player's death.
