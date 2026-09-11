---
theme: default
title: AI Coder
info: |
  ## AI Coder

  AI Day · InnoRAD.ai
class: text-center
colorSchema: light
fonts:
  sans: Jost
  serif: Jost
  mono: Geist Mono
  weights: '400,500,600,700,800'
drawings:
  persist: false
transition: fade
mdc: true
duration: 35min
clicks: 1
---
<div class="flex flex-col items-center justify-center h-full">
  <div class="text-8xl font-extrabold tracking-tight">AI Coder</div>
  <div class="rule-red mt-6"></div>
  <div class="text-2xl text-center mt-10 transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'" style="color:#71717a;max-width:44rem">
  An agentic coding suite for the AGIL Ops Hub platform, focused on the
  code stage.
  </div>
</div>

<div class="abs-br m-6 text-sm" style="color:#a1a1aa">
AI Day · InnoRAD.ai
</div>

<!--
TITLE

Say the name. Let it sit.

Click: the definition. An agentic coding suite for the AGIL Ops Hub
platform, focused on the code stage. Read it once, do not expand on it.
The next nine slides are the expansion.
-->

---
clicks: 1
class: flex flex-col justify-center items-center
---

# Where it fits

<div class="mt-10">
<div class="flex justify-center" style="width:48.0rem;gap:1.0rem">
<div class="rounded-lg border text-sm py-1.5 font-medium text-center" style="width:5.375rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Planning</div>
<div class="rounded-lg border text-sm py-1.5 font-medium text-center" style="width:5.375rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Requirements</div>
<div class="rounded-lg border text-sm py-1.5 font-medium text-center" style="width:5.375rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Design</div>
<div class="rounded-lg border text-sm py-1.5 font-medium text-center transition-all duration-700" :style="'width:16.125rem;' + ($clicks >= 1 ? 'border-color:#dc2626;background:rgba(220,38,38,0.08);color:#dc2626' : 'border-color:#d4d4d8;background:#fafafa;color:#18181b')">Implementation</div>
<div class="rounded-lg border text-sm py-1.5 font-medium text-center" style="width:5.375rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Testing</div>
<div class="rounded-lg border text-sm py-1.5 font-medium text-center" style="width:5.375rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Maintenance</div>
</div>
<div class="flex justify-center mt-2 transition-all duration-700" style="width:48.0rem;gap:1.0rem" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
<div class="rounded-lg border text-lg py-2 font-bold text-center" style="width:16.125rem;border-color:#dc2626;background:#dc2626;color:#ffffff">AI Coder</div>
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
</div>
<div class="flex justify-center mt-2 transition-all duration-700" style="width:48.0rem;gap:1.0rem" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
<div class="rounded-lg border text-xs py-1.5 font-medium text-center" style="width:16.125rem;border-color:rgba(220,38,38,0.25);background:rgba(220,38,38,0.05);color:#9f5f5f">Design principles &middot; Coding standards &middot; AGIL Ops Hub</div>
<div style="width:5.375rem"></div>
<div style="width:5.375rem"></div>
</div>
</div>

<!--
WHAT AI CODER IS

Be precise about what it is. The model is off the shelf. The tool that
runs it is off the shelf too. AI Coder is the layer we own.

Full definition if you need it: an agentic coding suite for the AGIL Ops
Hub platform, focused on the code stage. Underneath it is a swappable
coding harness, extended with agent skills and spec-driven development.

The band at the bottom of the diagram is what it carries: our design
principles, our coding standards, the platform.

That is the part worth having. Anyone can buy the same model next quarter.
Nobody can buy how MSS builds.

It also means we are not betting on a model. When a better one arrives,
our knowledge still applies.

The command is `aia init`, if you want to name it, but the room does
not need it.

Beat one: this is how we deliver, end to end. Build is the long pole.

Beat two, click: AI Coder assists the developer through the build, and it
arrives carrying our design principles, our coding standards, and the
AGIL Ops Hub platform.

Say "assists". The developer is still building. AI Coder is not replacing
anyone, and this audience will hear the difference.

The point for this room: nobody has to remember our standards or look them
up. They are in the project from the first minute, so the work starts
consistent instead of being corrected later.

Planning, test and deploy stay with the team.
-->

---
clicks: 1
class: flex flex-col justify-center items-center
---

# One command

<div class="rounded-xl px-8 py-6 text-left" style="width:46rem;background:#18181b">
<div class="flex items-center gap-2 mb-5">
<div style="width:0.7rem;height:0.7rem;border-radius:50%;background:#52525b"></div>
<div style="width:0.7rem;height:0.7rem;border-radius:50%;background:#52525b"></div>
<div style="width:0.7rem;height:0.7rem;border-radius:50%;background:#52525b"></div>
<div class="font-mono text-xs ml-3" style="color:#71717a">terminal</div>
</div>
<div class="font-mono text-lg">
<span style="color:#71717a">$ </span><span style="color:#fafafa">aia init</span></div>
<div class="font-mono text-sm mt-3"><span style="color:#71717a">? </span><span style="color:#a1a1aa">Project name</span><span class="pl-4" style="color:#fafafa">fleet-dispatch-console</span></div>
<div class="mt-4">
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#a1a1aa">Resolving template source</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#a1a1aa">Configuring OpenSpec workflows</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#a1a1aa">Initializing OpenSpec</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#a1a1aa">Installing agent skills</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#a1a1aa">Initializing git repository</span></div>
</div>
<div class="font-mono text-base font-bold mt-5" style="color:#fafafa">Project ready</div>
</div>

<div class="text-2xl text-center mt-8 transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'" style="color:#71717a;max-width:44rem">
It <strong>equips</strong> a new project with our way of building.
</div>

<!--
ONE COMMAND

This is the answer to "what is it". It is a command-line tool. Show the
terminal and let it speak.

The command is just `aia init`. It asks for the project name, so the name
is an answer, not part of the command.

Then the real steps, in the real order. It pulls the template, configures
the spec workflow, installs the twenty-three agent skills, and commits.
About twenty seconds.

Click for the line underneath. Equips is the word: the project arrives
already carrying how we build, so nobody has to remember it or look it up.

Do not say "nothing to install". There are prerequisites, and the room may
know it: Node, pnpm, Podman, OpenSpec, Git, make and Python. The CLI checks
them for you with `aia check` and tells you what is missing. What you do not
install is our standards, our skills or the workflow. Those arrive with the
project.

Be straight if asked what is not in there: the application. No app code,
no platform modules. The repository is not the product, it is everything
the product needs to be built our way.

This is exactly the first commit of the demo repository, pinned at
@mssfoobar/agent-skills v0.9.0. Run it today and you get the same tree.
-->

---
clicks: 1
class: flex flex-col justify-center items-center
---

# Then you describe the work

<div class="rounded-xl px-8 py-6 text-left" style="width:46rem;background:#18181b">
<div class="flex items-center gap-8 mb-6">
<div style="display:flex;flex-direction:column;gap:0">
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
</div>
<div class="font-mono text-sm" style="line-height:1.7">
<div><span style="color:#fafafa;font-weight:700">Claude Code</span> <span style="color:#71717a">v2.1.221</span></div>
<div style="color:#71717a">Opus 5 (1M context) with xhigh effort</div>
<div style="color:#71717a">~/Desktop/projects/ai-day-demo</div>
</div>
</div>
<div class="font-mono text-lg"><span style="color:#71717a">&gt; </span><span style="color:#fafafa">/opsx:propose</span></div>
<div class="font-mono text-sm mt-3" style="color:#d4d4d8;line-height:1.6">
As a dispatcher, I want to dispatch an available unit to an<br/>
incident and stand it down when the job is done.
</div>
<div class="mt-5">
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#fafafa;width:9rem">proposal.md</span><span style="color:#71717a">why, and what changes</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#fafafa;width:9rem">design.md</span><span style="color:#71717a">how it will be built</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#fafafa;width:9rem">specs/</span><span style="color:#71717a">36 requirements, 77 scenarios</span></div>
<div class="font-mono text-sm flex gap-3 mt-1.5"><span style="color:#4ade80">&check;</span><span style="color:#fafafa;width:9rem">tasks.md</span><span style="color:#71717a">82 steps, in order</span></div>
</div>
</div>

<div class="text-2xl text-center mt-8 transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'" style="color:#71717a;max-width:44rem">
One sentence in. A written plan out, <strong>before any code</strong>.
</div>

<!--
THEN YOU DESCRIBE THE WORK

This is the developer doing their actual job. They write the work the way
they already write it, as a user story, and hand it over.

That story is real, straight out of the demo repo. One sentence.

Out comes a written plan: why the change is needed, how it will be built,
the requirements with their testable scenarios, and the tasks in order.

The numbers are from the demo repo. Three stories became thirty-six
requirements, seventy-seven scenarios and eighty-two tasks.

Click: one sentence in, a written plan out, before any code. That is the
whole value of this step. Nothing has been built yet, so nothing has been
wasted if the plan is wrong.

Note the window title: this one runs inside Claude Code, not a shell.
The previous slide was a plain terminal. That is the honest sequence:
scaffold from the shell, then do the work inside the agent.

It is OpenSpec doing the spec work, an open-source tool, if anyone asks
by name.
-->

---
clicks: 3
class: flex flex-col justify-center items-center
---

# What you review

<div class="flex flex-col gap-2 mt-6" style="width:44rem">
<div class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 0 ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)' : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'">
<div class="flex items-center justify-between px-5 py-2.5">
<div class="font-mono text-sm font-bold transition-all duration-700 ease-in-out" :style="$clicks === 0 ? 'color:#dc2626' : 'color:#d4a0a0'">proposal.md</div>
<div class="text-[11px]" style="color:#a1a1aa">why are we doing this</div>
</div>
<div class="overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 0 ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'">
<div class="px-5 pb-4" style="height:9.5rem">
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">Why</span></div>
<div class="font-mono text-[13px] leading-relaxed" style="color:#3f3f46">Nothing ever writes to the database.</div>
<div class="font-mono text-[13px] leading-relaxed" style="color:#3f3f46">Persistence is only the seed.</div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">What Changes</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">dispatch-svc gains write endpoints</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">the console gains add, edit and delete</span></div>
</div>
</div>
</div>
<div class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 1 ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)' : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'">
<div class="flex items-center justify-between px-5 py-2.5">
<div class="font-mono text-sm font-bold transition-all duration-700 ease-in-out" :style="$clicks === 1 ? 'color:#dc2626' : 'color:#d4a0a0'">design.md</div>
<div class="text-[11px]" style="color:#a1a1aa">how it will be built</div>
</div>
<div class="overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 1 ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'">
<div class="px-5 pb-4" style="height:9.5rem">
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">Context</span></div>
<div class="font-mono text-[13px] leading-relaxed" style="color:#3f3f46">dispatch-svc is read-only and the console renders it.</div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">Goals / Non-Goals</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">create, replace and delete a unit from the console</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">prove persistence across a restart</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">not in scope: bulk edit, audit history</span></div>
</div>
</div>
</div>
<div class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 2 ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)' : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'">
<div class="flex items-center justify-between px-5 py-2.5">
<div class="font-mono text-sm font-bold transition-all duration-700 ease-in-out" :style="$clicks === 2 ? 'color:#dc2626' : 'color:#d4a0a0'">specs/</div>
<div class="text-[11px]" style="color:#a1a1aa">how we will know it works</div>
</div>
<div class="overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 2 ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'">
<div class="px-5 pb-4" style="height:9.5rem">
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">### </span><span style="color:#3f3f46;font-weight:700">Requirement: Add a unit from the console</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">#### </span><span style="color:#3f3f46">Scenario: Successful add</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">WHEN an operator submits the form with valid values</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">THEN the new unit appears in the list</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">#### </span><span style="color:#3f3f46">Scenario: Add rejected by validation</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">WHEN a blank required field or a duplicate code</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- </span><span style="color:#3f3f46">THEN the form stays open and shows the message</span></div>
</div>
</div>
</div>
<div class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 3 ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)' : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'">
<div class="flex items-center justify-between px-5 py-2.5">
<div class="font-mono text-sm font-bold transition-all duration-700 ease-in-out" :style="$clicks === 3 ? 'color:#dc2626' : 'color:#d4a0a0'">tasks.md</div>
<div class="text-[11px]" style="color:#a1a1aa">what happens, in what order</div>
</div>
<div class="overflow-hidden transition-all duration-700 ease-in-out" :style="$clicks === 3 ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'">
<div class="px-5 pb-4" style="height:9.5rem">
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">1. Implement dispatch-svc</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- [ ] </span><span style="color:#3f3f46">1.1 Consult the aoh-conventions skill</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- [ ] </span><span style="color:#3f3f46">1.2 Add OccLock to the unit model</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- [ ] </span><span style="color:#3f3f46">1.3 Repository: create, update, delete</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">## </span><span style="color:#3f3f46;font-weight:700">2. Implement dispatch-web</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- [ ] </span><span style="color:#3f3f46">2.1 Consult the aoh-conventions and aoh-design skills</span></div>
<div class="font-mono text-[13px] leading-relaxed"><span style="color:#c4c4c8">- [ ] </span><span style="color:#3f3f46">2.2 Add unit form, edit form, delete confirm</span></div>
</div>
</div>
</div>
</div>

<!--
WHAT YOU REVIEW

Four documents, one at a time. Click through them.

proposal.md: why. The problem and what changes because of it.

design.md: how. The approach, and what is deliberately out of scope.

specs/: how we will know. Point out that the second scenario is a failure
case. The plan states what happens when validation rejects the input, not
just when it works.

tasks.md: the order. Point at 1.1 and 2.1, both "consult the
aoh-conventions skill". The plan tells the AI to read our standards before
it writes a line. That is the platform knowledge being used, not just
installed.

The boxes are empty on purpose. Nothing has happened yet. This is the
plan, not a progress report.

Land it: a developer reads all four in a few minutes, and nothing is built
until they agree it.
-->

---
clicks: 2
class: flex flex-col justify-center items-center
---

# Then you let it run

<div class="rounded-xl px-8 py-6 text-left" style="width:46rem;background:#18181b">
<div class="flex items-center gap-8 mb-5">
<div style="display:flex;flex-direction:column;gap:0">
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#000000"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
<div style="display:flex;gap:0">
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
<div style="width:0.26rem;height:0.26rem;background:transparent"></div>
<div style="width:0.26rem;height:0.26rem;background:#d97757"></div>
</div>
</div>
<div class="font-mono text-sm" style="line-height:1.7">
<div><span style="color:#fafafa;font-weight:700">Claude Code</span></div>
<div style="color:#71717a">~/Desktop/projects/ai-day-demo</div>
</div>
</div>
<div class="font-mono text-lg mb-1"><span style="color:#71717a">&gt; </span><span style="color:#fafafa">/opsx:apply</span></div>
<div class="font-mono text-[13px] mt-3"><span style="color:#52525b">## </span><span style="color:#fafafa;font-weight:700">1. Implement dispatch-svc</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 1 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 1 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 1 ? 'color:#fafafa' : 'color:#71717a'">1.1 Consult the aoh-conventions skill</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 1 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 1 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 1 ? 'color:#fafafa' : 'color:#71717a'">1.2 Add OccLock to the unit model</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 1 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 1 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 1 ? 'color:#fafafa' : 'color:#71717a'">1.3 Repository: create, update, delete</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 1 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 1 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 1 ? 'color:#fafafa' : 'color:#71717a'">1.4 Service: validate fields, classify errors</span></div>
<div class="font-mono text-[13px] mt-3"><span style="color:#52525b">## </span><span style="color:#fafafa;font-weight:700">2. Implement dispatch-web</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 2 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 2 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 2 ? 'color:#fafafa' : 'color:#71717a'">2.1 Consult the aoh-conventions and aoh-design skills</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 2 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 2 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 2 ? 'color:#fafafa' : 'color:#71717a'">2.2 Add create, update and delete to the client</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 2 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 2 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 2 ? 'color:#fafafa' : 'color:#71717a'">2.3 Add unit form, edit form, delete confirm</span></div>
<div class="font-mono text-[13px] leading-relaxed transition-all duration-500"><span class="pr-2" :style="$clicks >= 2 ? 'color:#4ade80' : 'color:#52525b'">{{ $clicks >= 2 ? '- [x]' : '- [ ]' }} </span><span :style="$clicks >= 2 ? 'color:#fafafa' : 'color:#71717a'">2.4 Verify: build, lint, test</span></div>
<div class="font-mono text-sm mt-4 transition-all duration-500" :class="$clicks >= 2 ? 'opacity-100' : 'opacity-0'" style="color:#4ade80;font-weight:700">All 82 steps complete.</div>
</div>

<!--
THEN YOU LET IT RUN

The plan is agreed, so now it builds. Same checklist as the last slide,
ticking off.

Click through the two sections. The service first, then the console. Each
box ticks when that step is done, so the developer can see where it is
rather than waiting for a wall of output at the end.

This is the part that takes the time, and it is the part nobody has to
watch. The developer agreed the plan; the agent works through it.

Note 1.1 and 2.1 again: it consults our conventions before writing each
part. Not once at the start, but at the point it matters.

If asked what happens when a step fails: it stops and says so, against a
named step, rather than carrying on and reporting at the end.
-->

---
class: flex flex-col justify-center items-center
---

# Our stack

<div class="text-xl text-center mt-3 mb-10" style="color:#71717a;max-width:42rem">
The tool and the engine can change. The knowledge stays.
</div>

<div class="flex flex-col gap-3">
<div class="rounded-lg border-2 text-lg py-3 font-medium text-center" style="width:30.0rem;border-color:#dc2626;background:rgba(220,38,38,0.08);color:#dc2626">Coding agent</div>
<div class="rounded-lg border-2 text-lg py-3 font-bold text-center" style="width:30.0rem;border-color:#dc2626;background:#dc2626;color:#ffffff">AGIL Ops Hub knowledge and skills</div>
<div class="rounded-lg border-2 text-lg py-3 font-medium text-center" style="width:30.0rem;border-color:#dc2626;background:rgba(220,38,38,0.08);color:#dc2626">LLM</div>
</div>

<!--
OUR STACK

Three layers, top to bottom: the tool the developer works in, our
knowledge loaded into it, and the model underneath doing the writing.

The labels are deliberately generic. Today the coding agent is Claude Code
and the model is Claude, and say so if asked. They are named as categories
here because both are replaceable, and the slide would contradict itself if
it named a vendor.

Present it as one thing. This is the stack we run, and we built it to work
together.

The point of the subtitle: we are not tied to any vendor. A better tool
arrives, we swap the tool. A better model arrives, we swap the model. The
middle layer carries over every time, because it is ours.

That is worth saying plainly to this room. Choosing a stack is usually a
bet on a supplier. This one is not.

Straight answer if someone asks who makes the tool or the model: both are
licensed, chosen because they are the best available today and because
they are replaceable. What makes the stack ours is the middle layer and
the way the three are put together.
-->

---
class: flex flex-col justify-center items-center
---

# Why not just use an LLM?

<div class="flex gap-6 justify-center mt-10">
<div class="rounded-xl border-2 px-7 py-6 text-left" style="width:20.0rem;border-color:#d4d4d8;background:#fafafa">
<div class="text-lg font-bold mb-4" style="color:#18181b">An LLM on its own</div>
<div class="text-base mb-2.5" style="color:#71717a">Has never seen our code.</div>
<div class="text-base mb-2.5" style="color:#71717a">Fills in the blanks, confidently.</div>
<div class="text-base mb-2.5" style="color:#71717a">Nothing keeps it on course.</div>
<div class="text-base mb-2.5" style="color:#71717a">You only find out at the end.</div>
</div>
<div class="rounded-xl border-2 px-7 py-6 text-left" style="width:20.0rem;border-color:#dc2626;background:rgba(220,38,38,0.06)">
<div class="text-lg font-bold mb-4" style="color:#dc2626">AI Coder</div>
<div class="text-base mb-2.5" style="color:#18181b">Already knows how we build.</div>
<div class="text-base mb-2.5" style="color:#18181b">Keeps to the rules we set.</div>
<div class="text-base mb-2.5" style="color:#18181b">You agree the plan first.</div>
<div class="text-base mb-2.5" style="color:#18181b">You know where it is going.</div>
</div>
</div>

<!--
WHY NOT JUST AN LLM

This is the question the room is already holding. Answer it straight.

Be fair about it: an off-the-shelf LLM is fast, and the code often looks
fine. The problem is not speed. The problem is that nothing is steering.

It has never seen our codebase and it will not say so. It fills the gap
with something plausible, and it keeps going, whatever direction it took.

Third and fourth lines are the ones to dwell on. Left on its own, an LLM
runs the whole job before you see anything. You wait, you pay for the full
run, and only then do you learn it went the wrong way in the first ten
minutes.

AI Coder puts a checkpoint before the code. The plan is written down and
you agree it first, so a wrong turn costs a conversation instead of a
rewrite. That is OpenSpec doing the work, if anyone asks by name.

The point of the whole slide: the result is something we chose, not
something we were handed.

Do not oversell it. AI Coder does not remove review. It keeps the work
inside what we already decided, and moves the checking earlier, where it
is cheap.
-->

---
clicks: 1
class: flex flex-col justify-center
---

# How it works {.text-center}

<div class="mx-auto" style="width:51.00rem">
<div class="dgm-wide text-xs mt-8 mb-6" style="color:#a1a1aa;letter-spacing:0.08em">AI on its own</div>
<div class="flex items-center" style="width:51.00rem">
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:4.00rem;border-color:#e4e4e7;background:#fafafa;color:#71717a">Prompt</div>
<div class="text-center" style="width:1.0rem;color:#e4e4e7">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:16.00rem;border-color:#e4e4e7;background:#fafafa;color:#71717a">Develop</div>
<div class="text-center" style="width:1.0rem;color:#e4e4e7">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:12.00rem;border-color:#e4e4e7;background:#fafafa;color:#71717a">Verify</div>
<div class="text-center" style="width:1.0rem;color:#e4e4e7">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:16.00rem;border-color:#e4e4e7;background:#fafafa;color:#71717a">Redo</div>
</div>
<div class="flex items-start mt-1.5" style="width:51.00rem">
<div style="width:4.00rem"></div>
<div style="width:1.0rem"></div>
<div style="width:16.00rem"></div>
<div style="width:1.0rem"></div>
<div class="text-xs text-center " style="width:12.00rem;color:#a1a1aa">your only checkpoint</div>
<div style="width:1.0rem"></div>
<div class="text-xs text-center " style="width:16.00rem;color:#a1a1aa">and again</div>
</div>
<div class="flex" style="width:51.00rem">
<div style="width:2.00rem"></div>
<div style="width:41.00rem;height:1.4rem;border-left:2px solid #e4e4e7;border-right:2px solid #e4e4e7;border-bottom:2px solid #e4e4e7;border-radius:0 0 0.5rem 0.5rem"></div>
</div>
<div class="flex" style="width:51.00rem">
<div style="width:2.00rem"></div>
<div class="text-xs text-center mt-1" style="width:41.00rem;color:#a1a1aa">back to nothing</div>
</div>
<div class="transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">
<div class="dgm-wide text-xs mt-10 mb-6" style="color:#a1a1aa;letter-spacing:0.08em">With AI Coder</div>
<div class="flex items-center" style="width:43.00rem">
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:4.00rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Set up</div>
<div class="text-center" style="width:1.0rem;color:#d4d4d8">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:6.00rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Propose</div>
<div class="text-center" style="width:1.0rem;color:#d4d4d8">&rarr;</div>
<div class="rounded-lg border-2 text-sm py-2 font-bold text-center" style="width:4.00rem;border-color:#dc2626;background:#dc2626;color:#ffffff">Change</div>
<div class="text-center" style="width:1.0rem;color:#d4d4d8">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:16.00rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Develop</div>
<div class="text-center" style="width:1.0rem;color:#d4d4d8">&rarr;</div>
<div class="rounded-lg border-2 text-sm py-2 font-bold text-center" style="width:4.00rem;border-color:#dc2626;background:#dc2626;color:#ffffff">Verify</div>
<div class="text-center" style="width:1.0rem;color:#d4d4d8">&rarr;</div>
<div class="rounded-lg border text-sm py-2 font-medium text-center" style="width:4.00rem;border-color:#d4d4d8;background:#fafafa;color:#18181b">Archive</div>
</div>
<div class="flex items-start mt-1.5" style="width:43.00rem">
<div style="width:4.00rem"></div>
<div style="width:1.0rem"></div>
<div style="width:6.00rem"></div>
<div style="width:1.0rem"></div>
<div class="text-xs text-center font-bold" style="width:4.00rem;color:#dc2626">you</div>
<div style="width:1.0rem"></div>
<div style="width:16.00rem"></div>
<div style="width:1.0rem"></div>
<div class="text-xs text-center font-bold" style="width:4.00rem;color:#dc2626">you</div>
<div style="width:1.0rem"></div>
<div style="width:4.00rem"></div>
</div>
<div class="flex" style="width:43.00rem">
<div style="width:8.00rem"></div>
<div style="width:33.00rem;height:1.4rem;border-left:2px solid rgba(220,38,38,0.35);border-right:2px solid rgba(220,38,38,0.35);border-bottom:2px solid rgba(220,38,38,0.35);border-radius:0 0 0.5rem 0.5rem"></div>
</div>
<div class="flex" style="width:43.00rem">
<div style="width:8.00rem"></div>
<div class="text-xs text-center mt-1" style="width:33.00rem;color:#9f5f5f">next change</div>
</div>
</div>
</div>

<!--
HOW IT WORKS

Box width is time. Both rows start at the same point, so the length of the
row is how long the work takes.

Top: you prompt, it develops for a long stretch, you review the lot, and
if it went the wrong way you redo it. The loop goes back to nothing.

Click. Bottom: Develop is exactly as long. Nothing about the coding got
faster. What changed is everything around it.

Propose and Change are short, and they happen before a line is written.
Because you already agreed the plan, Verify is short too, and there is no
Redo block at all. The loop comes back to the next change, not to nothing.

Set up sits outside the loop. One command, once.

The honest claim: we are not making the AI faster. We are removing the
part you throw away.
-->
