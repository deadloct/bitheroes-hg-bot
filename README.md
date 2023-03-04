# Bit Heroes Squid Game

This Discord bot is activated by a player giving away a single friend spot in the
Kongregate game Bit Heroes. Players react to the pre-game message, which enters 
them into the competition. When the game starts, players are eliminated in a series
of rounds that follow the plot of the show Squid Game. Eventually one player wins
and the officiating player should give them a friend spot.

## Adding New Deaths to the Game

This bot reads the file `data/rounds.en.json` at startup. Deaths in this file should
use the generic Golang replacement string `%v` to represent the dying player's name.
Future versions will hopefully support multiple players or multiple mentions of
one player per string, but for now only one mention of the dying player is supported.
