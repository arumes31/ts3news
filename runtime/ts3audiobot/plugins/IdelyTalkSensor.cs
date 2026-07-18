// IdelyTalkSensor — a TS3AudioBot plugin that lets Idely know when a real user
// is talking in the bot's channel.
//
// Why this exists: TeamSpeak voice is channel-local and "talk status" is NOT a
// server notification — a client derives it from received voice packets. A
// headless client (like Idely's detector) has no audio device and never sees
// voice, so it cannot detect talking. TS3AudioBot, however, DOES decode audio in
// its channel. This plugin taps the bot's incoming-audio stream: every voice
// packet the bot receives carries the sender's client id (meta.In.Sender), and
// since voice is channel-local, any such packet means someone in the bot's
// channel is talking. The plugin records the time of the last voice packet and
// exposes it via the "idely voiceage" command, which Idely polls to decide when
// to stop and remove the bot.
//
// Drop this .cs into TS3AudioBot's plugins folder; the bot compiles it at load.

using System;
using System.Threading;
using TS3AudioBot.CommandSystem;
using TS3AudioBot.Plugins;
using TSLib.Audio;
using TSLib.Full;

public class IdelyTalkSensor : IBotPlugin, IAudioPassiveConsumer
{
	private readonly TsFullClient client;
	private IAudioPassiveConsumer next;
	private long lastVoiceTicks;

	// TsFullClient is the TSLib connection; it is the IAudioActiveProducer that
	// pushes received voice to its OutStream. Injected by the plugin loader.
	public IdelyTalkSensor(TsFullClient client)
	{
		this.client = client;
	}

	public void Initialize()
	{
		// Insert ourselves as a tap on the incoming-audio stream, forwarding to
		// whatever was there before so we don't disturb existing wiring.
		next = client.OutStream;
		client.OutStream = this;
	}

	public bool Active => next?.Active ?? true;

	public void Write(Span<byte> data, Meta meta)
	{
		// A non-whisper voice packet == someone talking in our channel.
		if (meta != null && !meta.In.Whisper)
			Interlocked.Exchange(ref lastVoiceTicks, DateTime.UtcNow.Ticks);
		next?.Write(data, meta);
	}

	// VoiceAge returns milliseconds since the last voice packet from the bot's
	// channel, or -1 if none has ever been heard. Idely polls this per bot.
	[Command("idely voiceage")]
	public string VoiceAge()
	{
		long t = Interlocked.Read(ref lastVoiceTicks);
		if (t == 0)
			return "-1";
		long ms = (DateTime.UtcNow.Ticks - t) / TimeSpan.TicksPerMillisecond;
		return ms.ToString();
	}

	public void Dispose()
	{
		// Restore the original stream on unload.
		if (ReferenceEquals(client.OutStream, this))
			client.OutStream = next;
	}
}
