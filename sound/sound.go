package sound

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hibooboo2/slime_tag/assets"

	// Alias the go-mp3 package to avoid conflict
	"github.com/hajimehoshi/ebiten/v2/audio/mp3" // Keep this import as is
	mp3lib "github.com/hajimehoshi/go-mp3"
)

var (
	audioContext *audio.Context
	// Export soundEffects to make it accessible from other packages
	soundEffects = make(map[string]*audio.Player)
)

func loadSound(filePath string) (*audio.Player, error) {
	f, err := assets.Resources.Open(filePath)
	if err != nil {
		return nil, err
	}
	//defer f.Close()

	// Use go-mp3 to read the sample rate
	mp3Decoder, err := mp3lib.NewDecoder(f) // Use the aliased package
	if err != nil {
		return nil, err
	}

	sampleRate := mp3Decoder.SampleRate() // Get the sample rate from the decoder

	// Reset the file pointer to the beginning for the Ebiten decoder
	//f.Seek(0, 0)

	// Use the mp3 decoder with the actual sample rate
	decoder, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, err
	}

	player, err := audioContext.NewPlayer(decoder)
	if err != nil {
		return nil, err
	}

	return player, nil
}

func init() {
	var err error
	audioContext = audio.NewContext(44100)

	soundEffects["hit"], err = loadSound("path/to/hit_sound.mp3")
	if err != nil {
		fmt.Println("Error loading hit sound:", err) // Log error
	}
	soundEffects["attack"], err = loadSound("resources/sound_effects/zapsplat_cartoon_slime_bubble_single_001_72385.mp3")
	if err != nil {
		fmt.Println("Error loading attack sound:", err) // Log error
	}
	soundEffects["death"], err = loadSound("resources/sound_effects/zapsplat_cartoon_whoosh_into_wet_slimy_impact_hit_003_48723.mp3")
	if err != nil {
		fmt.Println("Error loading death sound:", err) // Log error
	}
	soundEffects["headstone"], err = loadSound("resources/sound_effects/zapsplat_cartoon_slime_drip_single_003_50313.mp3")
	if err != nil {
		fmt.Println("Error loading headstone sound:", err) // Log error
	}
	soundEffects["powerattack"], err = loadSound("resources/sound_effects/zapsplat_cartoon_slime_drip_or_bubble_pop_002_65988.mp3")
	if err != nil {
		fmt.Println("Error loading powerup attack sound:", err) // Log error
	}
	soundEffects["healing powerup"], err = loadSound("resources/sound_effects/zapsplat_fantasy_dark_magic_whoosh_evil_spirit_or_demon_banish_slime_002_89592.mp3")
	if err != nil {
		fmt.Println("Error loading healing powerup attack sound:", err) // Log error
	}
	soundEffects["player damage"], err = loadSound("resources/sound_effects/zapsplat_science_fiction_impact_alien_body_hit_hard_crack_squelch_bloody_004_44158.mp3")
	if err != nil {
		fmt.Println("Error loading player damage sound:", err) // Log error
	}

	// Log loaded sounds
	fmt.Println("Loaded sounds:", soundEffects)
}

// Exported Play function to play sound effects
func Play(soundName string) error {
	player, exists := soundEffects[soundName]
	if !exists {
		return fmt.Errorf("sound %s not found", soundName)
	}
	player.Rewind() // Rewind to the beginning
	player.Play()   // Call Play() without returning its value
	return nil      // Return nil to indicate success
}
