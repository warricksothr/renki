using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;

namespace Renki.Tasks
{
    /// <summary>How a task ended.</summary>
    public sealed class Outcome
    {
        public string Status;
        public string Reason;
        public string Detail;
        public string Entity;

        public static Outcome Succeeded(string detail = null) =>
            new Outcome { Status = Wire.StatusSucceeded, Detail = detail };

        public static Outcome Failed(string reason, string detail = null) =>
            new Outcome { Status = Wire.StatusFailed, Reason = reason, Detail = detail };
    }

    /// <summary>A body task in progress. Tick runs once per physics step on
    /// the main thread and returns null while the task is still going.</summary>
    public interface IBodyTask
    {
        Outcome Tick(double dt);

        /// <summary>Stop now; no outcome is reported for a cancelled task.</summary>
        void Cancel();
    }

    /// <summary>Thrown by the host when a task cannot start, with the
    /// reason to report (invalid_target, unreachable, …).</summary>
    public sealed class TaskRefused : Exception
    {
        public string Reason { get; }

        public TaskRefused(string reason, string detail) : base(detail)
        {
            Reason = reason;
        }
    }

    /// <summary>What the runner needs from the game.</summary>
    public interface ITaskHost
    {
        /// <summary>Starts a body task, or throws TaskRefused.</summary>
        IBodyTask StartBody(TaskRequest req, object args);

        /// <summary>Runs an instant task to completion.</summary>
        Outcome RunInstant(TaskRequest req, object args);

        /// <summary>No body task is running any more: stand still.</summary>
        void Idle();
    }

    /// <summary>
    /// The task lifecycle (docs/BRIDGE-PROTOCOL.md 6.6 and 7.1) and the safety
    /// rules of section 8. It holds at most one body task; instant tasks run
    /// alongside it. Every task the controller sends gets a reply:
    /// rejected, or accepted followed by its later statuses.
    /// </summary>
    public sealed class TaskRunner
    {
        readonly SafetyConfig safety;
        readonly HashSet<string> implemented;
        readonly ITaskHost host;
        readonly Action<TaskStatus> report;
        readonly SayLimiter say;
        readonly Func<double> now;

        HashSet<string> allowed;
        TaskRequest current;
        IBodyTask body;
        double elapsed;

        public TaskRunner(SafetyConfig safety, IEnumerable<string> implemented, ITaskHost host,
            Action<TaskStatus> report, Func<double> now, SayLimiter say = null)
        {
            this.safety = safety;
            this.implemented = new HashSet<string>(implemented);
            this.host = host;
            this.report = report;
            this.now = now;
            this.say = say ?? new SayLimiter();
        }

        /// <summary>The tasks this build implements, for hello.capabilities.</summary>
        public IEnumerable<string> Capabilities => implemented.OrderBy(t => t, StringComparer.Ordinal);

        /// <summary>True between a welcome and the end of that session.</summary>
        public bool InSession => allowed != null;

        /// <summary>What may run in this session: implemented ∩ safety ∩ welcome.</summary>
        public IEnumerable<string> Allowed => allowed ?? Enumerable.Empty<string>();

        /// <summary>The running body task, for state.task.</summary>
        public TaskRef Current =>
            current == null ? null : new TaskRef { Id = current.Id, Task = current.Task, Status = Wire.StatusRunning };

        /// <summary>A new session: whatever ran before is stopped, and only
        /// tasks every layer allows may run.</summary>
        public void Welcome(IEnumerable<string> allowedTasks)
        {
            CancelBody(Wire.ReasonStopped, "new session", reportIt: false);
            allowed = new HashSet<string>(allowedTasks.Where(t => implemented.Contains(t) && safety.Permits(t)));
        }

        /// <summary>The dead-man rule: the controller has gone quiet. Stop,
        /// stand still, and accept nothing until the next welcome.</summary>
        public void EndSession(string detail)
        {
            CancelBody(Wire.ReasonStopped, detail, reportIt: false);
            allowed = null;
        }

        public void Handle(TaskRequest req)
        {
            if (!implemented.Contains(req.Task))
            {
                Reply(req, Wire.StatusRejected, Wire.ReasonUnsupported, $"this plugin does not do {req.Task}");
                return;
            }
            if (!safety.Permits(req.Task))
            {
                Reply(req, Wire.StatusRejected, Wire.ReasonNotAllowed, "the plugin's safety config forbids it");
                return;
            }
            if (allowed == null || !allowed.Contains(req.Task))
            {
                Reply(req, Wire.StatusRejected, Wire.ReasonNotAllowed, "not in this session's allowedTasks");
                return;
            }
            object args;
            try
            {
                args = TaskArgs.Parse(req);
            }
            catch (BadArgsException e)
            {
                Reply(req, Wire.StatusRejected, Wire.ReasonBadArgs, e.Message);
                return;
            }

            if (req.Task == TaskNames.Stop)
            {
                Reply(req, Wire.StatusAccepted);
                CancelBody(Wire.ReasonStopped, "stop", reportIt: true);
                Reply(req, Wire.StatusSucceeded);
                return;
            }

            if (TaskNames.IsInstant(req.Task))
            {
                if (args is SayArgs s)
                {
                    s.Text = SayLimiter.Clean(s.Text);
                    if (s.Text.Length == 0)
                    {
                        Reply(req, Wire.StatusRejected, Wire.ReasonBadArgs, "nothing left to say");
                        return;
                    }
                    if (!say.TryTake(now()))
                    {
                        Reply(req, Wire.StatusRejected, Wire.ReasonNotAllowed, "say is rate limited");
                        return;
                    }
                }
                Reply(req, Wire.StatusAccepted);
                Outcome o;
                try
                {
                    o = host.RunInstant(req, args);
                }
                catch (TaskRefused e)
                {
                    o = Outcome.Failed(e.Reason, e.Message);
                }
                Reply(req, o.Status, o.Reason, o.Detail);
                return;
            }

            IBodyTask started;
            try
            {
                started = host.StartBody(req, args);
            }
            catch (TaskRefused e)
            {
                Reply(req, Wire.StatusRejected, e.Reason, e.Message);
                return;
            }
            CancelBody(Wire.ReasonSuperseded, $"replaced by {req.Id}", reportIt: true);
            current = req;
            body = started;
            elapsed = 0;
            Reply(req, Wire.StatusAccepted);
            Reply(req, Wire.StatusRunning);
        }

        /// <summary>Advances the body task by one physics step.</summary>
        public void FixedTick(double dt)
        {
            if (body == null) return;
            elapsed += dt;
            Outcome o;
            if (current.TimeoutS > 0 && elapsed > current.TimeoutS)
            {
                body.Cancel();
                o = Outcome.Failed(Wire.ReasonTimeout, $"{current.TimeoutS:0.#} s elapsed");
            }
            else
            {
                o = body.Tick(dt);
                if (o == null) return;
            }
            var finished = current;
            current = null;
            body = null;
            host.Idle();
            Reply(finished, o.Status, o.Reason, o.Detail, o.Entity);
        }

        /// <summary>The bot died: the body task fails with <c>died</c>.</summary>
        public void Died()
        {
            if (body == null) return;
            var finished = current;
            body.Cancel();
            current = null;
            body = null;
            host.Idle();
            Reply(finished, Wire.StatusFailed, Wire.ReasonDied, null);
        }

        void CancelBody(string reason, string detail, bool reportIt)
        {
            if (body == null)
            {
                host.Idle();
                return;
            }
            var cancelled = current;
            body.Cancel();
            current = null;
            body = null;
            host.Idle();
            if (reportIt) Reply(cancelled, Wire.StatusCancelled, reason, detail);
        }

        void Reply(TaskRequest req, string status, string reason = null, string detail = null, string entity = null) =>
            report(new TaskStatus { Re = req.Id, Status = status, Reason = reason, Detail = detail, Entity = entity });
    }
}
