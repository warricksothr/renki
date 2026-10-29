using System;
using System.Collections.Generic;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace Renki.Protocol
{
    /// <summary>Which side wrote a frame. <c>task</c> is a request from the
    /// controller and a status report from the plugin.</summary>
    public enum Sender { Plugin, Controller }

    /// <summary>Why a frame could not be decoded.</summary>
    public enum DecodeFailure { Malformed, Version, UnknownType }

    public sealed class ProtocolException : Exception
    {
        public DecodeFailure Failure { get; }

        /// <summary>The frame's <c>id</c>, when it had one, so an error reply can name it.</summary>
        public string FrameId { get; }

        public ProtocolException(DecodeFailure failure, string message, string frameId = null, Exception inner = null)
            : base(message, inner)
        {
            Failure = failure;
            FrameId = frameId;
        }
    }

    /// <summary>Decodes and encodes frames. Unknown fields are ignored; an
    /// unknown major version or type is refused (section 3).</summary>
    public static class Codec
    {
        static readonly Dictionary<string, Func<Message>> FromPlugin = new Dictionary<string, Func<Message>>
        {
            [Wire.Hello] = () => new Hello(),
            [Wire.State] = () => new State(),
            [Wire.Perception] = () => new Perception(),
            [Wire.Inventory] = () => new Inventory(),
            [Wire.Container] = () => new ContainerSnapshot(),
            [Wire.Layout] = () => new Layout(),
            [Wire.Chat] = () => new Chat(),
            [Wire.Event] = () => new Event(),
            [Wire.Task] = () => new TaskStatus(),
            [Wire.Pong] = () => new Pong(),
            [Wire.Bye] = () => new Bye(),
            [Wire.Error] = () => new Error(),
            [Wire.Trace] = () => new Trace(),
        };

        static readonly Dictionary<string, Func<Message>> FromController = new Dictionary<string, Func<Message>>
        {
            [Wire.Welcome] = () => new Welcome(),
            [Wire.Task] = () => new TaskRequest(),
            [Wire.Ping] = () => new Ping(),
            [Wire.Config] = () => new Config(),
            [Wire.Logout] = () => new Logout(),
            [Wire.Bye] = () => new Bye(),
            [Wire.Error] = () => new Error(),
        };

        static readonly JsonSerializerSettings Settings = new JsonSerializerSettings
        {
            MissingMemberHandling = MissingMemberHandling.Ignore,
            FloatParseHandling = FloatParseHandling.Double,
            Culture = System.Globalization.CultureInfo.InvariantCulture,
        };

        static readonly JsonSerializer Serializer = JsonSerializer.Create(Settings);

        public static Message Decode(string frame, Sender from)
        {
            JObject obj;
            try
            {
                obj = JObject.Parse(frame);
            }
            catch (JsonException e)
            {
                throw new ProtocolException(DecodeFailure.Malformed, "not a JSON object: " + e.Message, null, e);
            }
            var id = (string)obj["id"];
            var v = obj["v"];
            if (v == null || v.Type != JTokenType.Integer || (int)v != Wire.Version)
            {
                throw new ProtocolException(DecodeFailure.Version,
                    $"unsupported protocol version {v?.ToString() ?? "(none)"} (this side speaks {Wire.Version})", id);
            }
            var type = (string)obj["type"] ?? "";
            var table = from == Sender.Plugin ? FromPlugin : FromController;
            if (!table.TryGetValue(type, out var make))
            {
                throw new ProtocolException(DecodeFailure.UnknownType,
                    $"unknown message type \"{type}\" from the {from.ToString().ToLowerInvariant()}", id);
            }
            var msg = make();
            try
            {
                using (var reader = obj.CreateReader())
                {
                    Serializer.Populate(reader, msg);
                }
            }
            catch (JsonException e)
            {
                throw new ProtocolException(DecodeFailure.Malformed, $"decode {type}: {e.Message}", id, e);
            }
            return msg;
        }

        /// <summary>Writes msg as one frame, stamping <c>v</c> and <c>type</c>.</summary>
        public static string Encode(Message msg)
        {
            msg.V = Wire.Version;
            msg.Type = msg.MessageType;
            return JsonConvert.SerializeObject(msg, Formatting.None, Settings);
        }
    }
}
