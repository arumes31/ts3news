package content

import "math/rand"

// trivia holds short gaming trivia / fun facts appended to the bottom of the PM.
var trivia = []string{
	"The best-selling video game of all time is Minecraft, with over 300 million copies sold.",
	"Pac-Man was inspired by a pizza with a slice missing.",
	"The first known video game, 'Tennis for Two', was made in 1958 on an oscilloscope.",
	"Mario was originally called 'Jumpman' and was a carpenter, not a plumber.",
	"The Konami Code is: Up, Up, Down, Down, Left, Right, Left, Right, B, A.",
	"'Game over' screens exist because early arcade games needed players to insert more coins.",
	"The PlayStation started as a CD add-on for the Super Nintendo.",
	"Steam launched in 2003 and now hosts over 50,000 games.",
	"The longest video game marathon lasted more than 138 hours.",
	"Tetris was created by Russian engineer Alexey Pajitnov in 1984.",
	"The Sims is the best-selling PC franchise of all time.",
	"Sonic the Hedgehog's shoes were inspired by Michael Jackson's boots.",
	"Halo was originally going to be a real-time strategy game for Mac.",
	"The word 'avatar' in gaming comes from a 1985 Ultima game.",
	"GTA V cost roughly $265 million to make and market.",
	"The Nintendo Wii outsold the PlayStation 3 for most of its lifespan.",
	"Doom (1993) is so portable it 'runs on everything' — including pregnancy tests and fridges.",
	"The first Easter egg was hidden in Atari's 'Adventure' (1980).",
	"World of Warcraft once had over 12 million subscribers at its peak.",
	"The Game Boy survived a bombing in the Gulf War and still works at the Nintendo NY store.",
	"Link from Zelda is left-handed in most classic games.",
	"CD Projekt Red started as a company that translated games into Polish.",
	"The original Xbox was nearly named 'DirectX Box'.",
	"Minecraft was first built by Markus 'Notch' Persson in just six days.",
	"Final Fantasy was named because it was meant to be its studio's last game.",
	"The highest-grossing arcade game ever is Pac-Man.",
	"Counter-Strike began as a mod for Half-Life.",
	"The PlayStation 2 is the best-selling console of all time.",
	"League of Legends has more lines of voiced dialogue than most movies have scripts.",
	"Skyrim's iconic dragon shouts were performed by real choirs.",
	"The Witcher series is based on Polish novels by Andrzej Sapkowski.",
	"Portal's 'Still Alive' was written by Jonathan Coulton in a single weekend.",
	"Pong was almost rejected for being 'too simple'.",
	"The Legend of Zelda was the first console game to let players save progress.",
	"Kratos from God of War was voiced by the same actor across many games for over a decade.",
	"Animal Crossing runs in real time, synced to your console's clock.",
	"Red Dead Redemption 2 has over 500,000 lines of dialogue.",
	"E.T. for the Atari was so bad that unsold cartridges were buried in a New Mexico landfill.",
	"The first esports tournament was held at Stanford in 1972 for the game Spacewar.",
	"Super Mario Bros. was once thought to be unbeatable without dying — speedrunners now finish it in under 5 minutes.",
}

// RandomTrivia returns a random gaming trivia fact.
func RandomTrivia() string {
	return trivia[rand.Intn(len(trivia))]
}

// TriviaCount is exposed for tests.
func TriviaCount() int { return len(trivia) }
