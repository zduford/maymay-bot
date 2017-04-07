/*
 ************************ MAYMAY-BOT ************************
 ************************ By zduford ************************
 *********** http://github.com/zduford/maymay-bot ***********
 *********** Based on hammerandchisel's airhornbot **********
 ************************** ENJOY ***************************
 */

package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	queues map[string]chan *Play = make(map[string]chan *Play)

	// Sound encoding settings
	BITRATE        = 128
	MAX_QUEUE_SIZE = 6

	// Owner
	OWNER string

	//Version
	VERSION_RELEASE = "0.0.4-b"

	//TIME Constant
	t0 = time.Now()

	COUNT int = 0

	// Shard (or -1)
	SHARDS []string = make([]string, 0)

	mem runtime.MemStats
)

// Play represents an individual use of the meme sounds commands
type Play struct {
	GuildID   string
	ChannelID string
	UserID    string
	Sound     *Sound

	// The next play to occur after this, only used for chaining sounds like anotha
	Next *Play

	// If true, this was a forced play using a specific meme sound name
	Forced bool
}

type SoundCollection struct {
	Prefix    string
	Commands  []string
	Sounds    []*Sound
	ChainWith *SoundCollection

	soundRange int
}

// Sound represents a sound clip
type Sound struct {
	Name string

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int

	// Delay (in milliseconds) for the bot to wait before sending the disconnect request
	PartDelay int

	// Buffer to store encoded PCM packets
	buffer [][]byte
}

// Array of all the sounds we have
var DAMN *SoundCollection = &SoundCollection{
	Prefix: "damn",
	Commands: []string{
		"!damn",
	},

	Sounds: []*Sound{
		createSound("classic", 1000, 250),
	},
}

var DEEZNUTZ *SoundCollection = &SoundCollection{
	Prefix: "deezNuts",
	Commands: []string{
		"!deez",
		"!deeznutz",
	},
	Sounds: []*Sound{
		createSound("classic", 1000, 250),
	},
}

var HITMARKER *SoundCollection = &SoundCollection{
	Prefix: "hitMarker",
	Commands: []string{
		"!hitmarker",
	},
	Sounds: []*Sound{
		createSound("classic", 200, 250),
	},
}

var MMMSAY *SoundCollection = &SoundCollection{
	Prefix: "mmmsay",
	Commands: []string{
		"!whatcha",
		"!mmmsay",
	},
	Sounds: []*Sound{
		createSound("classic", 1000, 250),
	},
}

var SCREAM *SoundCollection = &SoundCollection{
	Prefix: "scream",
	Commands: []string{
		"!wilhelm",
		"!scream",
	},
	Sounds: []*Sound{
		createSound("classic", 1000, 250),
	},
}

var WOW *SoundCollection = &SoundCollection{
	Prefix: "wow",
	Commands: []string{
		"!wow",
	},
	Sounds: []*Sound{
		createSound("classic", 100, 250),
	},
}

var TRIPLE *SoundCollection = &SoundCollection{
	Prefix: "triple",
	Commands: []string{
		"!ohbaby",
		"!triple",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var ILLKILLYOU *SoundCollection = &SoundCollection{
	Prefix: "kill",
	Commands: []string{
		"!illkillyou",
		"!killyou",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var YOUDIP *SoundCollection = &SoundCollection{
	Prefix: "dip",
	Commands: []string{
		"!youdip",
		"!dip",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var RUBY *SoundCollection = &SoundCollection{
	Prefix: "ruby",
	Commands: []string{
		"!ruby",
		"!rubymadness",
	},
	Sounds: []*Sound{
		createSound("classic", 100, 250),
		createSound("classic2", 100, 250),
		createSound("cheeky", 100, 250),
		createSound("scream", 100, 250),
		createSound("scream2", 100, 250),
		createSound("longscream", 100, 250),
	},
}

var DEDODATED *SoundCollection = &SoundCollection{
	Prefix: "dedodated",
	Commands: []string{
		"!wam",
		"!dedodatedwam",
		"!dedodated",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var CLIPVIOLIN *SoundCollection = &SoundCollection{
	Prefix: "clipviolin",
	Commands: []string{
		"!clipviolin",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var TROMBONE *SoundCollection = &SoundCollection{
	Prefix: "trombone",
	Commands: []string{
		"!trombone",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var VIOLIN *SoundCollection = &SoundCollection{
	Prefix: "violin",
	Commands: []string{
		"!violin",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var WEED *SoundCollection = &SoundCollection{
	Prefix: "weed",
	Commands: []string{
		"!weed",
		"!snoop",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var LYIN *SoundCollection = &SoundCollection{
	Prefix: "lyin",
	Commands: []string{
		"!lyin",
	},
	Sounds: []*Sound{
		createSound("classic", 10, 250),
	},
}

var COLLECTIONS []*SoundCollection = []*SoundCollection{
	DAMN,
	DEEZNUTZ,
	HITMARKER,
	MMMSAY,
	SCREAM,
	WOW,
	TRIPLE,
	ILLKILLYOU,
	YOUDIP,
	RUBY,
	DEDODATED,
	CLIPVIOLIN,
	TROMBONE,
	VIOLIN,
	WEED,
	LYIN,
}

// Create a Sound struct
func createSound(Name string, Weight int, PartDelay int) *Sound {
	return &Sound{
		Name:      Name,
		Weight:    Weight,
		PartDelay: PartDelay,
		buffer:    make([][]byte, 0),
	}
}

func (sc *SoundCollection) Load() {
	for _, sound := range sc.Sounds {
		sc.soundRange += sound.Weight
		sound.Load(sc)
	}
}

func (s *SoundCollection) Random() *Sound {
	var (
		i      int
		number int = randomRange(0, s.soundRange)
	)

	for _, sound := range s.Sounds {
		i += sound.Weight

		if number < i {
			return sound
		}
	}
	return nil
}

// Load attempts to load an encoded sound file from disk
// DCA files are pre-computed sound files that are easy to send to Discord.
// If you would like to create your own DCA files, please use:
// https://github.com/nstafie/dca-rs
// eg: dca-rs --raw -i <input wav file> > <output file>
func (s *Sound) Load(c *SoundCollection) error {
	log.Info("Starting Thing")
	path := fmt.Sprintf("audio/%v_%v.dca", c.Prefix, s.Name)

	file, err := os.Open(path)

	if err != nil {
		fmt.Println("error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		// read opus frame length from dca file
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}

		if err != nil {
			fmt.Println("error reading from dca file :", err)
			return err
		}

		// read encoded pcm from dca file
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("error reading from dca file :", err)
			return err
		}

		// append encoded pcm data to the buffer
		s.buffer = append(s.buffer, InBuf)
	}
}

// Plays this sound over the specified VoiceConnection
func (s *Sound) Play(vc *discordgo.VoiceConnection) {
	vc.Speaking(true)
	defer vc.Speaking(false)

	for _, buff := range s.buffer {
		vc.OpusSend <- buff
	}
}

// Attempts to find the current users voice channel inside a given guild
func getCurrentVoiceChannel(user *discordgo.User, guild *discordgo.Guild) *discordgo.Channel {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			channel, _ := discord.State.Channel(vs.ChannelID)
			return channel
		}
	}
	return nil
}

// Returns a random integer between min and max
func randomRange(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}

// Prepares a play
func createPlay(user *discordgo.User, guild *discordgo.Guild, coll *SoundCollection, sound *Sound) *Play {
	// Grab the users voice channel
	channel := getCurrentVoiceChannel(user, guild)
	if channel == nil {
		log.WithFields(log.Fields{
			"user":  user.ID,
			"guild": guild.ID,
		}).Warning("Failed to find channel to play sound in")
		return nil
	}

	// Create the play
	play := &Play{
		GuildID:   guild.ID,
		ChannelID: channel.ID,
		UserID:    user.ID,
		Sound:     sound,
		Forced:    true,
	}

	// If we didn't get passed a manual sound, generate a random one
	if play.Sound == nil {
		play.Sound = coll.Random()
		play.Forced = false
	}

	// If the collection is a chained one, set the next sound
	if coll.ChainWith != nil {
		play.Next = &Play{
			GuildID:   play.GuildID,
			ChannelID: play.ChannelID,
			UserID:    play.UserID,
			Sound:     coll.ChainWith.Random(),
			Forced:    play.Forced,
		}
	}

	return play
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, coll *SoundCollection, sound *Sound) {
	play := createPlay(user, guild, coll, sound)
	if play == nil {
		return
	}

	// Check if we already have a connection to this guild
	//   yes, this isn't threadsafe, but its "OK" 99% of the time
	_, exists := queues[guild.ID]

	if exists {
		if len(queues[guild.ID]) < MAX_QUEUE_SIZE {
			queues[guild.ID] <- play
		}
	} else {
		queues[guild.ID] = make(chan *Play, MAX_QUEUE_SIZE)
		playSound(play, nil)
	}
}

func trackSoundStats(play *Play) {

}

// Play a sound
func playSound(play *Play, vc *discordgo.VoiceConnection) (err error) {
	log.WithFields(log.Fields{
		"play": play,
	}).Info("Playing sound")

	if vc == nil {
		vc, err = discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, false, false)
		// vc.Receive = false
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to play sound")
			delete(queues, play.GuildID)
			return err
		}
	}

	// If we need to change channels, do that now
	if vc.ChannelID != play.ChannelID {
		vc.ChangeChannel(play.ChannelID, false, false)
		time.Sleep(time.Millisecond * 125)
	}

	// Track stats for this play in redis
	go trackSoundStats(play)

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(time.Millisecond * 32)

	// Play the sound
	play.Sound.Play(vc)

	// If this is chained, play the chained sound
	if play.Next != nil {
		playSound(play.Next, vc)
	}

	// If there is another song in the queue, recurse and play that
	if len(queues[play.GuildID]) > 0 {
		play := <-queues[play.GuildID]
		playSound(play, vc)
		return nil
	}

	// If the queue is empty, delete it
	time.Sleep(time.Millisecond * time.Duration(play.Sound.PartDelay))
	delete(queues, play.GuildID)
	vc.Disconnect()
	return nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	s.UpdateStatus(0, "Buko No Pico")
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable == true {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			log.Info("Entered a new server")
			s.ChannelMessageSend(channel.ID, "**LITTLE GAY BOY**")
			return
		}
	}
}

func scontains(key string, options ...string) bool {
	for _, item := range options {
		if item == key {
			return true
		}
	}
	return false
}

func calculateAirhornsPerSecond(cid string) {
}

func displayBotStats(cid string) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	users := 0
	for _, guild := range discord.State.Ready.Guilds {
		users += len(guild.Members)
	}

	t1 := time.Now()
	d := t1.Sub(t0)
	minutesPassed := d.Minutes()
	var truncate int = int(minutesPassed) % 60
	var hoursPassed int = int(minutesPassed / 60)

	w := &tabwriter.Writer{}
	buf := &bytes.Buffer{}

	w.Init(buf, 0, 4, 0, ' ', 0)
	fmt.Fprintf(w, "```\n")
	fmt.Fprintf(w, "Discordgo: \t%s\n", discordgo.VERSION)
	fmt.Fprintf(w, "Go: \t%s\n", runtime.Version())
	fmt.Fprintf(w, "maymay-bot ver.: \t%s\n", VERSION_RELEASE)
	fmt.Fprintf(w, "Time Up: \t%v hrs. %v min.\n", hoursPassed, truncate)
	fmt.Fprintf(w, "Memory: \t%d / %d (%d total)\n", mem.Alloc, mem.Sys, mem.TotalAlloc)
	fmt.Fprintf(w, "Calls: \t%d\n", COUNT)
	fmt.Fprintf(w, "Servers: \t%d\n", len(discord.State.Ready.Guilds))
	fmt.Fprintf(w, "```\n")
	w.Flush()
	discord.ChannelMessageSend(cid, buf.String())
}

func utilSumRedisKeys(keys []string) int {
	return 0
}

func displayUserStats(cid, uid string) {
}

func displayServerStats(cid, sid string) {

}

func utilGetMentioned(s *discordgo.Session, m *discordgo.MessageCreate) *discordgo.User {
	for _, mention := range m.Mentions {
		if mention.ID != s.State.Ready.User.ID {
			return mention
		}
	}
	return nil
}

func airhornBomb(cid string, guild *discordgo.Guild, user *discordgo.User, cs string) {
	count, _ := strconv.Atoi(cs)
	discord.ChannelMessageSend(cid, ":ok_hand:"+strings.Repeat(":trumpet:", count))

	// Cap it at something
	if count > 100 {
		return
	}

	play := createPlay(user, guild, HITMARKER, nil)
	vc, err := discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, true, true)
	if err != nil {
		return
	}

	for i := 0; i < count; i++ {
		HITMARKER.Random().Play(vc)
	}

	vc.Disconnect()
}

// Handles bot operator messages, should be refactored (lmao)
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
	if scontains(parts[1], "status") {
		displayBotStats(m.ChannelID)
	} else if scontains(parts[1], "stats") {
		if len(m.Mentions) >= 2 {
			displayUserStats(m.ChannelID, utilGetMentioned(s, m).ID)
		} else if len(parts) >= 3 {
			displayUserStats(m.ChannelID, parts[2])
		} else {
			displayServerStats(m.ChannelID, g.ID)
		}
	} else if scontains(parts[1], "bomb") && len(parts) >= 4 {
		airhornBomb(m.ChannelID, g, utilGetMentioned(s, m), parts[3])
	} else if scontains(parts[1], "aps") {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go calculateAirhornsPerSecond(m.ChannelID)
	} else if scontains(parts[len(parts)-1], "info") && m.Author.ID == OWNER {
		runtime.ReadMemStats(&mem)
		t1 := time.Now()
		d := t1.Sub(t0)
		minutesPassed := d.Minutes()
		var truncate int = int(minutesPassed) % 60
		var hoursPassed int = int(minutesPassed / 60)
		w := &tabwriter.Writer{}
		buf := &bytes.Buffer{}

		w.Init(buf, 0, 4, 0, ' ', 0)
		fmt.Fprintf(w, "```\n")
		fmt.Fprintf(w, "Discordgo: \t%s\n", discordgo.VERSION)
		fmt.Fprintf(w, "Go: \t%s\n", runtime.Version())
		fmt.Fprintf(w, "maymay-bot ver.: \t%s\n", VERSION_RELEASE)
		fmt.Fprintf(w, "Time Up: \t%v hrs. %v min.\n", hoursPassed, truncate)
		fmt.Fprintf(w, "Memory: \t%d / %d (%d total)\n", mem.Alloc, mem.Sys, mem.TotalAlloc)
		fmt.Fprintf(w, "Calls: \t%d\n", COUNT)
		fmt.Fprintf(w, "Servers: \t%d\n", len(discord.State.Ready.Guilds))
		fmt.Fprintf(w, "```\n")
		w.Flush()
		s.ChannelMessageSend(m.ChannelID, buf.String())
	} else if scontains(parts[len(parts)-1], "where") {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("its a me, shard %v", string(g.ID[len(g.ID)-5])))
	} else if scontains(parts[len(parts)-1], "killbot") && m.Author.ID == OWNER {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: goodbye cruel world")
		os.Exit(0)
	}
	return
}

func generateCommandList() string {
	var commands string
	commands = "`Here's some of those dank sounds you meme loving fuck \n\n"
	commands = commands + "!damn\n!deez\n!hitmarker\n!mmmsay\n!scream\n!wow\n!triple\n!illkillyou\n"
	commands = commands + "!dip\n!ruby\n!dedodated\n!trombone\n!violin\n!weed\n!lyin\n!roll\n\n"
	commands = commands + "Now go out there and make me proud you autists.`"
	return commands
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Content) <= 0 || (m.Content[0] != '!' && len(m.Mentions) < 1) {
		return
	}

	msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	parts := strings.Split(strings.ToLower(msg), " ")

	channel, _ := discord.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := discord.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	// If this is a mention, it should come from the owner (otherwise we don't care)
	if len(m.Mentions) > 0 && m.Author.ID == OWNER && len(parts) > 0 {
		mentioned := false
		for _, mention := range m.Mentions {
			mentioned = (mention.ID == s.State.Ready.User.ID)
			if mentioned {
				break
			}
		}

		//if mentioned {
		//handleBotControlMessages(s, m, parts, guild)
		//}
		return
	}
	log.Info(parts[0])
	// If !commands is sent
	if parts[0] == "!commands" {
		log.Info("I tried to send commands")
		log.Info(channel.ID)
		COUNT++
		var commands string
		commands = generateCommandList()
		mess, err := s.ChannelMessageSend(channel.ID, commands)
		log.Info(mess)
		log.Info(err)
		return
	}

	if parts[0] == "!roll" {
		COUNT++
		re := regexp.MustCompile("^[0-9]*$")
		if len(parts) == 1 {
			var num = randomRange(1, 20)
			s.ChannelMessageSend(channel.ID, "```Rolling d20```")
			time.Sleep(time.Millisecond * 100)
			s.ChannelMessageSend(channel.ID, fmt.Sprintf("```%v```", num))
			return
		} else {
			var amt int = 1
			var splitD = strings.Split(parts[1], "d")
			//if a command like 2d6
			if re.MatchString(splitD[0]) { //checking if [1] is a num
				amt, _ = strconv.Atoi(splitD[0])
				if amt > 5 && m.Author.ID != OWNER { //Allows the owner to be a spammy jerk
					s.ChannelMessageSend(channel.ID, "```Whoa there buddy, only 5 at a time```")
					return
				}
			}

			if splitD[0] == parts[1] {
				s.ChannelMessageSend(channel.ID, "```Invalid entry, try 'd20' or 'd6'```")
				return
			}

			if re.MatchString(splitD[1]) { //if [1] is not a num
			} else {
				s.ChannelMessageSend(channel.ID, "```Invalid entry, try 'd20' or 'd6'```")
				return
			}

			if splitD[1] == "" {
				s.ChannelMessageSend(channel.ID, "```Invalid entry, try 'd20' or 'd6'```")
				return
			}
			var max int
			max, _ = strconv.Atoi(splitD[1])
			var num int
			if amt == 0 {
				amt++
			}
			if max == 0 {
				s.ChannelMessageSend(channel.ID, fmt.Sprintf("```Fuck You Dylan```"))
				return
			}
			for i := 0; i < amt; i++ {
				num = randomRange(1, max+1)
				s.ChannelMessageSend(channel.ID, fmt.Sprintf("```Rolling d%v```", max))
				time.Sleep(time.Millisecond * 50)
				s.ChannelMessageSend(channel.ID, fmt.Sprintf("```%v```", num))
			}
			return
		}

	}

	// Find the collection for the command we got
	for _, coll := range COLLECTIONS {
		if scontains(parts[0], coll.Commands...) {

			// If they passed a specific sound effect, find and select that (otherwise play nothing)
			var sound *Sound
			if len(parts) > 1 {
				for _, s := range coll.Sounds {
					if parts[1] == s.Name {
						sound = s
					}
				}

				if sound == nil {
					return
				}
			}

			go enqueuePlay(m.Author, guild, coll, sound)
			return
		}
	}
}

func main() {
	var (
		Token      = flag.String("t", "", "Discord Authentication Token")
		Shard      = flag.String("s", "", "Shard ID")
		ShardCount = flag.String("c", "", "Number of shards")
		Owner      = flag.String("o", "", "Owner ID")
		err        error
	)
	flag.Parse()

	if *Owner != "" {
		OWNER = *Owner
	}

	// Preload all the sounds
	log.Info("Preloading sounds...")
	for _, coll := range COLLECTIONS {
		coll.Load()
	}

	// If we got passed a redis server, try to connect
	//lol nope

	// Create a discord session
	log.Info("Starting discord session...")
	discord, err = discordgo.New("Bot " + *Token)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	// Set sharding info
	discord.ShardID, _ = strconv.Atoi(*Shard)
	discord.ShardCount, _ = strconv.Atoi(*ShardCount)

	if discord.ShardCount <= 0 {
		discord.ShardCount = 1
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}

	// We're running!
	log.Info("MEMEBOT is ready to autist it up.")

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
