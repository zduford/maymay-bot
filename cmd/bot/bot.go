package main

import (
        "encoding/binary"
        "flag"
        "fmt"
        "io"
        "math/rand"
        "os"
        "os/exec"
        "os/signal"
        "strconv"
        "strings"
        "time"
        
        log "github.com/Sirupsen/logrus"
        "github.com/bwmarrin/discordgo"
        "github.com/layeh/gopus"
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
     
     // Shard (or -1)
     SHARDS []string = make([]string, 0)
     )

// Play represents an individual use of the !airhorn command
type Play struct {
    GuildID   string
    ChannelID string
    UserID    string
    Sound     *Sound
    
    // The next play to occur after this, only used for chaining sounds like anotha
    Next *Play
    
    // If true, this was a forced play using a specific airhorn sound name
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
    
    // Channel used for the encoder routine
    encodeChan chan []int16
    
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
Prefix:    "deezNuts",
    //ChainWith: AIRHORN,
Commands: []string{
    "!deez",
    "!deezNutz",
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
    createSound("classic", 1000, 250),
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
    createSound("classic", 1000, 250),
},
}

var TRIPLE *SoundCollection = &SoundCollection{
Prefix: "triple",
Commands: []string{
    "!ohbaby",
    "!triple",
},
Sounds: []*Sound{
    createSound("classic", 1000, 250),
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
}

// Create a Sound struct
func createSound(Name string, Weight int, PartDelay int) *Sound {
    return &Sound{
    Name:       Name,
    Weight:     Weight,
    PartDelay:  PartDelay,
    encodeChan: make(chan []int16, 10),
    buffer:     make([][]byte, 0),
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

// Encode reads data from ffmpeg and encodes it using gopus
func (s *Sound) Encode() {
    encoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
    if err != nil {
        fmt.Println("NewEncoder Error:", err)
        return
    }
    
    encoder.SetBitrate(BITRATE * 1000)
    encoder.SetApplication(gopus.Audio)
    
    for {
        pcm, ok := <-s.encodeChan
        if !ok {
            // if chan closed, exit
            return
        }
        
        // try encoding pcm frame with Opus
        opus, err := encoder.Encode(pcm, 960, 960*2*2)
        if err != nil {
            fmt.Println("Encoding Error:", err)
            return
        }
        
        // Append the PCM frame to our buffer
        s.buffer = append(s.buffer, opus)
    }
}

// Load attempts to load and encode a sound file from disk
func (s *Sound) Load(c *SoundCollection) error {
    s.encodeChan = make(chan []int16, 10)
    defer close(s.encodeChan)
    go s.Encode()
    
    path := fmt.Sprintf("audio/%v_%v.wav", c.Prefix, s.Name)
    ffmpeg := exec.Command("ffmpeg", "-i", path, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
    
    stdout, err := ffmpeg.StdoutPipe()
    if err != nil {
        fmt.Println("StdoutPipe Error:", err)
        return err
    }
    
    err = ffmpeg.Start()
    if err != nil {
        fmt.Println("RunStart Error:", err)
        return err
    }
    
    for {
        // read data from ffmpeg stdout
        InBuf := make([]int16, 960*2)
        err = binary.Read(stdout, binary.LittleEndian, &InBuf)
        
        // If this is the end of the file, just return
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            return nil
        }
        
        if err != nil {
            fmt.Println("error reading from ffmpeg stdout :", err)
            return err
        }
        
        // write pcm data to the encodeChan
        s.encodeChan <- InBuf
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

// Whether a guild id is in this shard
func shardContains(guildid string) bool {
    if len(SHARDS) != 0 {
        ok := false
        for _, shard := range SHARDS {
            if len(guildid) >= 5 && string(guildid[len(guildid)-5]) == shard {
                ok = true
                break
            }
        }
        return ok
    }
    return true
}

// Returns a random integer between min and max
func randomRange(min, max int) int {
    rand.Seed(time.Now().UTC().UnixNano())
    return rand.Intn(max-min) + min
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, coll *SoundCollection, sound *Sound) {
    // Grab the users voice channel
    channel := getCurrentVoiceChannel(user, guild)
    if channel == nil {
        log.WithFields(log.Fields{
                       "user":  user.ID,
                       "guild": guild.ID,
                       }).Warning("Failed to find channel to play sound in")
        return
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
    s.UpdateStatus(0, "Kill Yourself")
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
    if !shardContains(event.Guild.ID) {
        return
    }
    
    if event.Guild.Unavailable != nil {
        return
    }
    
    for _, channel := range event.Guild.Channels {
        if channel.ID == event.Guild.ID {
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


// Handles bot operator messages, should be refactored (lmao)
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
    ourShard := shardContains(g.ID)
    if len(parts) >= 3 && scontains(parts[len(parts)-2], "die") {
        shard := parts[len(parts)-1]
        if len(SHARDS) == 0 || scontains(shard, SHARDS...) {
            log.Info("Got DIE request, exiting...")
            s.ChannelMessageSend(m.ChannelID, ":ok_hand: goodbye cruel world")
            os.Exit(0)
        }
    } else if scontains(parts[len(parts)-1], "aps") && ourShard {
        s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
    } else if scontains(parts[len(parts)-1], "where") && ourShard {
        s.ChannelMessageSend(m.ChannelID,
                             fmt.Sprintf("its a me, shard %v", string(g.ID[len(g.ID)-5])))
    }
    return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    if len(m.Content) <= 0 || (m.Content[0] != '!' && len(m.Mentions) != 1) {
        return
    }
    
    parts := strings.Split(strings.ToLower(m.Content), " ")
    
    channel, _ := discord.State.Channel(m.ChannelID)
    if channel == nil {
        log.WithFields(log.Fields{
                       "channel": m.ChannelID,
                       "message": m.ID,
                       }).Warning("Failed to grab channel")
        //return
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
    if len(m.Mentions) > 0 {
        if m.Mentions[0].ID == s.State.Ready.User.ID && m.Author.ID == OWNER && len(parts) > 0 {
            handleBotControlMessages(s, m, parts, guild)
        }
        return
    }
    
    // If it's not relevant to our shard, just exit
    if !shardContains(guild.ID) {
        return
    }
    
    // If !commands is sent
    if parts[0] == "!commands" {
        s.ChannelMessageSend(channel.ID, "`To Do AKA fUCK yOU`")
        return
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
         Token = flag.String("t", "", "Discord Authentication Token")
         Shard = flag.String("s", "", "Integers to shard by")
         Owner = flag.String("o", "", "Owner ID")
         err   error
         )
    flag.Parse()
    
    if *Owner != "" {
        OWNER = *Owner
    }
    
    // Make sure shard is either empty, or an integer
    if *Shard != "" {
        SHARDS = strings.Split(*Shard, ",")
        
        for _, shard := range SHARDS {
            if _, err := strconv.Atoi(shard); err != nil {
                log.WithFields(log.Fields{
                               "shard": shard,
                               "error": err,
                               }).Fatal("Invalid Shard")
                return
            }
        }
    }
    
    // Preload all the sounds
    log.Info("Preloading sounds...")
    for _, coll := range COLLECTIONS {
        coll.Load()
    }
    
    
    // Create a discord session
    log.Info("Starting discord session...")
    discord, err = discordgo.New(*Token)
    if err != nil {
        log.WithFields(log.Fields{
                       "error": err,
                       }).Fatal("Failed to create discord session")
        return
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