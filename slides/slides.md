---
theme: default
title: AI Coder
info: |
  ## AI Coder

  AI Day · InnoRAD.ai
class: flex flex-col justify-center items-center
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
defaults:
  class: flex flex-col justify-center items-center
---

# Contents

<Points>
<NumStep :n="1" pad="py-1">Background</NumStep>
<NumStep :n="2" pad="py-1">AI Coder</NumStep>
<NumStep :n="3" pad="py-1">How it works</NumStep>
<SubPoint>AGIL Ops Hub</SubPoint>
<SubPoint>Agent Skills</SubPoint>
<SubPoint>OpenSpec</SubPoint>
<SubPoint>Components</SubPoint>
<NumStep :n="4" pad="py-1">Workshop</NumStep>
<NumStep :n="5" pad="py-1">Build your own</NumStep>
</Points>

<!--
CONTENTS

Thirty seconds. Read the five, do not expand on any of them.

The only one worth a beat is three: that is the part most of the room has
not seen before. Four is the demo and what to install. Five is how they
make the AI coder their own.
-->

---

# Background

<Points>
<Bullet>Build a C2 application, as fast as AI could take us.</Bullet>
<Bullet>A C2 application needs a <span class="key">huge amount of detail</span>.</Bullet>
<Bullet>Whatever you leave out, the model <span class="key">fills in for you</span>.</Bullet>
<Bullet>And you find out only <span class="key">after</span> it is finished.</Bullet>
</Points>

<!--
Background

So mid last year, we were tasked to explore how fast we can build a C2 application quickly using AI. During the research, we discovered 2 main problems;

- Context
Prompting for a full fledged application is extremely tedious. This is due to the scale of a C2 application; It requires an exorbident amount of details in the specifications, and missing them, tend to have the model to fill up what it is not specified (which might not be whats prefered).

Which leads to the next problem where you realised whats wrong only after it is done.
-->

---

# Two questions

<Points>
<NumStep :n="1">Can we keep the results <span class="key">consistent and grouded to our platform</span> </NumStep>
<NumStep :n="2">Can we pin down what it builds, <span class="key">before</span> it builds it?</NumStep>
</Points>

<!--
THE TWO QUESTIONS

Two questions come out of that.

First, consistency. Every developer feeds the model their own version of the
detail, so every result comes back a little different. Can the standards live
in the platform instead, so the same request gives the same shape of answer
whoever is asking?

Second, verification. Right now the model fills the gaps its own way and you
find out at the end. Can we pin the work down before it starts, and correct it
there instead?

These two questions are what the rest of the deck answers.
-->

---

```yaml
clicks: 1
class: text-center
```

<div class="flex flex-col items-center justify-center h-full">
  <div class="text-8xl font-extrabold tracking-tight">AI Coder</div>
  <div class="rule-red mt-6"></div>
  <div class="text-2xl text-center mt-10 transition-all duration-700 text-muted max-w-[44rem]" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">
  A tool that generates C2 application with AGIL Ops Hub Platform
  </div>
</div>

<!--
AI Coder is a coding tool that generates C2 applications quickly using the AGIL Ops Hub platform.
-->

---

# AGIL Ops Hub

<Points>
<Bullet>A platform contains a set of modules, tools and services commonly used by C2 applications</Bullet>
<Bullet>Agent is to build C2 applications <span class="key">on top of</span> the AGIL Ops Hub platform.</Bullet>
<Bullet>Maintain a certain degree of <span class="key">deterministic behaviour</span>.</Bullet>
</Points>

<!--
AGIL OPS HUB

Yours to write. Worth covering, based on what the deck already claims:

- what AGIL Ops Hub is, in one sentence
- that it is the middle layer of the Components diagram coming up, the red box
- what is actually in it: the knowledge, the skills, the conventions
- who maintains it and how a team gets changes into it
- why it carries over when the harness or the model is swapped
-->

---

# Agent Skills

<Points>
<Bullet>A package of instructions scripts and contexts that an Agent can discover and read.</Bullet>
<Bullet>Only knows what skills are available and only reads it only <span class="key">at the exact point of work</span></Bullet>
<Bullet>Reduces context memory on start.</Bullet>
<Bullet>Converted AGIL Ops Hub platform knowledge into Agent Skills </Bullet>
</Points>

<!--
SKILLS

This is the red box of the Components diagram coming up, on its own.

A skill is not a prompt. It is our standards written down in a form the
agent reads by itself: how we code, how we design, how we review.

There is one per discipline, and the twenty-three of them arrive with the
project rather than being pasted in per developer.

The line that matters is the third one. The agent consults a skill at the
step where it is relevant, not once at the beginning and then forgotten.
Point forward to the task list later in the deck: step 1.1 and step 2.1 both
read "consult the aoh-conventions skill", because that is where it matters.

That is the answer to the first of the two questions. The standards are the
same for everyone because nobody is retyping them.
-->

---

# OpenSpec (Spec-driven Development) {.text-center}

<Points>
<Bullet>Framework to enable <span class="key">spec-driven development</span></Bullet>
<Bullet v-click="1">Practice of a written specifications of your artifact</Bullet>
<Bullet v-click="1">Treating code as generated from these specifications</Bullet>
</Points>

<!--
Explain definition

The idea isn't new. What spec-driven used to be like was like this.

*Display loop*

What changed is the economics. When a human wrote every line, the spec was overhead you paid once and then abandoned as the code drifted. Now that generating code is cheap and reviewing it is the bottleneck, the expensive scarce thing is a precise statement of intent. Agents will produce something plausible for any prompt you give them, including a badly underspecified one, so vagueness doesn't fail loudly. It fails as confident, working code that solves the wrong problem.
-->

---

# Components

<ArchDiagram class="mt-6" />

<!--
COMPONENTS

This is the wrap-up of the three you have just seen, not an introduction to
them. Name each piece as you point at it and move on.

Start with the dashed box. Everything inside it runs on the developer's own
machine. Only the model sits outside, and that is the only thing that leaves.

The developer machine has the CLI installed. It hydrates the project with the
context and knowledge of the AGIL Ops Hub platform.

Click. That is what it installs: the skills the room just saw, and OpenSpec.

Click. The coding harness reads those artifacts and feeds them to an LLM,
cloud or on-prem. That is what makes it consistently aware of the domain
knowledge before it writes anything.

The harness and the model are both swappable. The red box is the part that
is ours, and it carries over every time.
-->

---

```yaml
clicks: 6
```

# How a change gets built {.text-center}

<div class="flex items-start justify-center gap-10 mt-4">
<div class="flex flex-col">
<FlowColumn
  :steps="['Story', 'Propose spec', 'Review spec', 'Implement', 'Review build', 'Ship', 'Archive']"
  :active="$clicks"
/>
</div>

<!-- The artefacts only exist while the spec is being proposed and reviewed. -->
<SpecStack
  class="flex flex-col gap-2 w-[34rem] transition-all duration-700"
  :class="$clicks === 1 || $clicks === 2 ? 'opacity-100' : 'opacity-0'"
>
<SpecCard
  name="proposal.md"
  caption="why are we doing this"
  :active="true"
  :lines="[
    '## Why',
    'Nothing ever writes to the database.',
    'Persistence is only the seed.',
    '## What Changes',
    '- dispatch-svc gains write endpoints',
    '- the console gains add, edit and delete',
  ]"
/>
<SpecCard
  name="design.md"
  caption="how it will be built"
  :active="false"
  :lines="[
    '## Context',
    'dispatch-svc is read-only and the console renders it.',
    '## Goals / Non-Goals',
    '- create, replace and delete a unit from the console',
    '- prove persistence across a restart',
    '- not in scope: bulk edit, audit history',
  ]"
/>
<SpecCard
  name="specs/"
  caption="how we will know it works"
  :active="false"
  :lines="[
    '### Requirement: Add a unit from the console',
    '#### Scenario: Successful add',
    '- WHEN an operator submits the form with valid values',
    '- THEN the new unit appears in the list',
    '#### Scenario: Add rejected by validation',
    '- WHEN a blank required field or a duplicate code',
    '- THEN the form stays open and shows the message',
  ]"
/>
<SpecCard
  name="tasks.md"
  caption="what happens, in what order"
  :active="false"
  :lines="[
    '## 1. Implement dispatch-svc',
    '- [ ] 1.1 Consult the aoh-conventions skill',
    '- [ ] 1.2 Add OccLock to the unit model',
    '- [ ] 1.3 Repository: create, update, delete',
    '## 2. Implement dispatch-web',
    '- [ ] 2.1 Consult the aoh-conventions and aoh-design skills',
    '- [ ] 2.2 Add unit form, edit form, delete confirm',
  ]"
/>
</SpecStack>
</div>

<!--
HOW A CHANGE GETS BUILT

The column on the left is the whole flow, top to bottom. One click per stage,
and the red outline is wherever you are.

Story is yours. Click.

Propose spec. The four documents appear on the right: why, how, how we will
know, and in what order. This is what the agent hands back. Click.

Review spec. Same four documents, because this is the stage where you read
them. This is the gate: nothing is implemented until it passes. If the spec is
wrong you send it back, and that costs a spec rather than a build. Click.

Implement, approve the build, ship. The documents are gone from the screen
because the arguing is over.

Archive is the last click, and it is bookkeeping rather than building. The
change folded its deltas back into the permanent specs, so the next story
starts from a spec that describes what the system actually does now. Then the
loop comes back to the next story.

If someone asks in the room: the four documents are clickable. Click one to
open it and the others close, so you can jump straight to whichever they ask
about instead of clicking through in order.
-->

---

```yaml
clicks: 1
```

# Whats the Difference? {.text-center}

<div class="flex items-start justify-center gap-20 mt-10">
<LoopDiagram
  label="Coding with an agent"
  :nodes="['Prompt', 'Build', 'Review']"
  :accent="2"
/>
<div v-click class="transition-all duration-700">
<LoopDiagram
  variant="brand"
  label="Spec-driven development"
  :nodes="['Story', 'Spec', 'Review', 'Build']"
  :accent="2"
/>
</div>
</div>

<!--
WHERE YOU CHECK THE WORK

Both rings start at the top and run clockwise, so trace each one with a
finger rather than reading the boxes out.

Left: prompt, it builds, then you review. The first time a human gets a say
is after the code exists, so a wrong turn costs the whole build.

Click. Right: story, spec, you review, then it builds. The gate comes first,
so nothing is written until you have agreed the plan, and a wrong turn costs
a spec.

The people icon marks the gate in both. On the left it is the last stop
before you start over. On the right it sits ahead of the build.

That answers the second question from earlier. You see what it plans to do
before it does the work, and you correct it there, not at the end.
-->

---

<SectionTitle>Workshop</SectionTitle>

<!--
WORKSHOP

Section marker. Everything so far was the argument. From here it is the
tool doing the work, and then what they need to install before the session.
-->

---

# Setting up

<Points>
<NumStep :n="1">Install Node 24+, Go 1.25+ and Python 3</NumStep>
<NumStep :n="2">Install Podman, and start its machine</NumStep>
<NumStep :n="3" body="font-mono text-sm">npm install -g pnpm@10</NumStep>
<NumStep :n="4" body="font-mono text-sm">curl -fsSL https://claude.ai/install.sh | bash</NumStep>
<NumStep :n="5" body="font-mono text-sm">cd ai-day-demo && pnpm start</NumStep>
<NumStep :n="6">Open http://127.0.0.1.nip.io:5173/aoh/dispatch/units</NumStep>
</Points>

<!--
SETTING UP

The only slide where falling behind costs someone the session, so walk it.

Everything runs natively. Only PostgreSQL is a container, which is what Podman
is for.

Step two matters on macOS and Windows: Podman Desktop needs its machine started
before any container will run. Ask the room to open it now.

Python is not used by the project. It is there for the scripts the agent writes
during the exercises. macOS and Linux already have it; Windows people need the
installer, and need to tick "Add python.exe to PATH".

Steps three and four are the two that are typed rather than downloaded. Say the
@10 out loud: unpinned gives pnpm 12, which our lockfile was not written for.
Windows PowerShell runs step four as: irm https://claude.ai/install.ps1 | iex

After step four they need a fresh terminal before claude is on PATH. Tell them
now, or a third of the room reports command not found.

No token to set up. The GitHub Packages credential for our six dependencies,
including the design system, is checked into .npmrc in the repo.

pnpm start installs, pulls PostgreSQL, compiles the Go service, then runs. A
few minutes the first time, under a second after. Safe to retype if anything
goes wrong.

That first run is the one that needs internet. It fills node_modules, the Go
module cache and the Podman image store, and all three survive going offline.

If someone already runs Postgres on 5432, the runner says so by name and they
start again with POSTGRES_PORT set to a free port.
-->
---

```yaml
clicks: 1
```

# Then you describe the work

<TermPanel>
<ClaudeHeader pad="mb-6">
<div><span class="text-term-fg font-bold">Claude Code</span> <span class="text-term-dim">v2.1.221</span></div>
<div class="text-term-dim">Opus 5 (1M context) with xhigh effort</div>
<div class="text-term-dim">~/Desktop/projects/ai-day-demo</div>
</ClaudeHeader>
<PromptLine>/opsx:propose</PromptLine>
<div class="font-mono text-sm mt-3 text-steel leading-[1.6]">
As a dispatcher, I want to dispatch an available unit to an<br/>
incident and stand it down when the job is done.
</div>
<div class="mt-5">
<CheckLine name="proposal.md">why, and what changes</CheckLine>
<CheckLine name="design.md">how it will be built</CheckLine>
<CheckLine name="specs/">36 requirements, 77 scenarios</CheckLine>
<CheckLine name="tasks.md">82 steps, in order</CheckLine>
</div>
</TermPanel>

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

```yaml
clicks: 3
```

# What you review

<SpecStack class="flex flex-col gap-2 mt-6 w-[44rem]">
<SpecCard
  name="proposal.md"
  caption="why are we doing this"
  :active="$clicks === 0"
  :lines="[
    '## Why',
    'Nothing ever writes to the database.',
    'Persistence is only the seed.',
    '## What Changes',
    '- dispatch-svc gains write endpoints',
    '- the console gains add, edit and delete',
  ]"
/>
<SpecCard
  name="design.md"
  caption="how it will be built"
  :active="$clicks === 1"
  :lines="[
    '## Context',
    'dispatch-svc is read-only and the console renders it.',
    '## Goals / Non-Goals',
    '- create, replace and delete a unit from the console',
    '- prove persistence across a restart',
    '- not in scope: bulk edit, audit history',
  ]"
/>
<SpecCard
  name="specs/"
  caption="how we will know it works"
  :active="$clicks === 2"
  :lines="[
    '### Requirement: Add a unit from the console',
    '#### Scenario: Successful add',
    '- WHEN an operator submits the form with valid values',
    '- THEN the new unit appears in the list',
    '#### Scenario: Add rejected by validation',
    '- WHEN a blank required field or a duplicate code',
    '- THEN the form stays open and shows the message',
  ]"
/>
<SpecCard
  name="tasks.md"
  caption="what happens, in what order"
  :active="$clicks === 3"
  :lines="[
    '## 1. Implement dispatch-svc',
    '- [ ] 1.1 Consult the aoh-conventions skill',
    '- [ ] 1.2 Add OccLock to the unit model',
    '- [ ] 1.3 Repository: create, update, delete',
    '## 2. Implement dispatch-web',
    '- [ ] 2.1 Consult the aoh-conventions and aoh-design skills',
    '- [ ] 2.2 Add unit form, edit form, delete confirm',
  ]"
/>
</SpecStack>

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

# Then you let it run

<TermPanel>
<ClaudeHeader pad="mb-5">
<div><span class="text-term-fg font-bold">Claude Code</span></div>
<div class="text-term-dim">~/Desktop/projects/ai-day-demo</div>
</ClaudeHeader>
<PromptLine pad="mb-1">/opsx:apply</PromptLine>
<div class="font-mono text-[13px] mt-3"><span class="text-term-dimmer">## </span><span class="text-term-fg font-bold">1. Implement dispatch-svc</span></div>
<TaskLine :done="$clicks >= 1">1.1 Consult the aoh-conventions skill</TaskLine>
<TaskLine :done="$clicks >= 1">1.2 Add OccLock to the unit model</TaskLine>
<TaskLine :done="$clicks >= 1">1.3 Repository: create, update, delete</TaskLine>
<TaskLine :done="$clicks >= 1">1.4 Service: validate fields, classify errors</TaskLine>
<div class="font-mono text-[13px] mt-3"><span class="text-term-dimmer">## </span><span class="text-term-fg font-bold">2. Implement dispatch-web</span></div>
<TaskLine :done="$clicks >= 2">2.1 Consult the aoh-conventions and aoh-design skills</TaskLine>
<TaskLine :done="$clicks >= 2">2.2 Add create, update and delete to the client</TaskLine>
<TaskLine :done="$clicks >= 2">2.3 Add unit form, edit form, delete confirm</TaskLine>
<TaskLine :done="$clicks >= 2">2.4 Verify: build, lint, test</TaskLine>
<div class="font-mono text-sm mt-4 font-bold text-term-ok transition-all duration-500" :class="$clicks >= 2 ? 'opacity-100' : 'opacity-0'">All 82 steps complete.</div>
</TermPanel>

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

```yaml
clicks: 1
```

# One command

<TermPanel>
<TermDots />
<PromptLine sigil="$">aia init</PromptLine>
<div class="font-mono text-sm mt-3"><span class="text-term-dim">? </span><span class="text-faint">Project name</span><span class="pl-4 text-term-fg">fleet-dispatch-console</span></div>
<div class="mt-4">
<CheckLine>Resolving template source</CheckLine>
<CheckLine>Configuring OpenSpec workflows</CheckLine>
<CheckLine>Initializing OpenSpec</CheckLine>
<CheckLine>Installing agent skills</CheckLine>
<CheckLine>Initializing git repository</CheckLine>
</div>
<div class="font-mono text-base font-bold mt-5 text-term-fg">Project ready</div>
</TermPanel>

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

<SectionTitle>Build your own</SectionTitle>

<!--
BUILD YOUR OWN

Section marker. Everything so far used our AI coder. This part shows that it
is made of plain files in the repository, and that they can write their own.
-->

---

```yaml
clicks: 3
```

# Three things you write

<div class="flex flex-col items-center gap-3 mt-8">
<div class="transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">
<StackLayer>AGENTS.md <span class="font-normal opacity-70">· read every session</span></StackLayer>
</div>
<div class="transition-all duration-700" :class="$clicks >= 2 ? 'opacity-100' : 'opacity-0'">
<StackLayer>Agent skills <span class="font-normal opacity-70">· read when the task needs them</span></StackLayer>
</div>
<div class="transition-all duration-700" :class="$clicks >= 3 ? 'opacity-100' : 'opacity-0'">
<StackLayer>OpenSpec schema <span class="font-normal opacity-70">· the steps every change takes</span></StackLayer>
</div>
</div>

<!--
THREE THINGS YOU WRITE

None of this is code. It is Markdown and YAML in the repository, reviewed and
versioned like code. That is the point of the section: they can write it.

Click. AGENTS.md: the rules every session reads before it does anything. This
repo keeps it at the root. CLAUDE.md is a one-line import of it, so Claude Code
and the other tools that read AGENTS.md all get the same rules.

Click. Agent skills: know-how the agent reads only when the task calls for it.
We ship twenty-three for the platform.

Click. The OpenSpec schema: which documents every change produces, in what
order, and what each must contain.

The next slides take the last two in turn.
-->

---

```yaml
clicks: 2
```

# Anatomy of a skill

<SpecStack class="flex flex-col gap-2 mt-6 w-[44rem]">
<SpecCard
  name="SKILL.md"
  caption="when to use it, and how"
  :active="$clicks === 0"
  :lines="[
    '---',
    'name: aoh-gis-integration',
    'description: Integrate the AOH GIS module. Use when a user wants a map.',
    '---',
    '### Step 1. Opt the map route out of SSR',
    '### Step 8. Styles',
  ]"
/>
<SpecCard
  name="references/"
  caption="detail, read when a step needs it"
  :active="$clicks === 1"
  :lines="[
    '- components.md',
    '- entities.md',
    '- auth-and-config.md',
    '- bff-bookmarks.md',
  ]"
/>
<SpecCard
  name="scripts/"
  caption="things it runs instead of rewriting"
  :active="$clicks === 2"
  :lines="['- seed-geoentities.sh']"
/>
</SpecStack>

<!--
ANATOMY OF A SKILL

A skill is a folder. This is the real GIS skill from this repository, cut down
to fit.

SKILL.md: a name, a description, then the steps. The description is abridged
here; the real one lists the phrases people use when they want a map.

Click. references/: the detail. The agent opens these only when a step needs
them, so a long reference costs nothing until it is used.

Click. scripts/: things the agent runs rather than working out again each time.
Here, seeding test entities onto the map.

The part to write most carefully is the description. Next slide.
-->

---

```yaml
class: text-center
```

<div class="flex flex-col items-center justify-center h-full">
  <div class="text-6xl font-bold tracking-tight">The description is the trigger</div>
  <div class="rule-red mt-6"></div>
  <div class="text-2xl text-muted mt-10 max-w-[44rem]">Write it in the words people will ask with</div>
</div>

<!--
THE DESCRIPTION IS THE TRIGGER

Only the name and description of each skill sit in the agent's context all the
time, for all twenty-nine skills at once. The body loads when the agent decides
the task matches the description.

So a skill with a vague description never loads, however good its body is.

Write the description the way a developer phrases the request. Ours list them
outright: "add a map", "integrate GIS", "render entities on a map".
-->

---

# A mistake, written down once

<SpecCard
  class="mt-6 w-[44rem]"
  name="apps/dispatch-web/AGENTS.md"
  caption="added while preparing this workshop"
  :active="true"
  :lines="[
    '### Tailwind only scans this app because src/app.css says so',
    '- It also imports @mssfoobar/gis-web-sdk/styles/app.css.',
    '- Do not remove that either.',
    '- Without it the marker overlay loses its z-20: the pin renders under the map.',
  ]"
/>

<!--
A MISTAKE, WRITTEN DOWN ONCE

A real one. While preparing exercise two, the inline map rendered but the unit
pin did not.

The app never imported the GIS SDK stylesheet, so one CSS class the pin relies
on had no rule, and the pin sat underneath the map. The fix was a single line.

These lines, abridged from the real file, are what stops the next agent from
deleting it again. That is the habit to leave with: when the agent gets
something wrong, write down why, where the next agent will read it.

When the lesson is about the platform rather than this app, it belongs in the
platform skill, here aoh-gis-integration, so every project gets it.
-->

---

```yaml
class: text-center
```

<div class="flex flex-col items-center justify-center h-full">
  <div class="text-6xl font-bold tracking-tight">Test it cold</div>
  <div class="rule-red mt-6"></div>
  <div class="text-2xl text-muted mt-10 max-w-[44rem]">A fresh agent, only the skill, a real project</div>
</div>

<!--
TEST IT COLD

Reading a skill does not catch the two failures that matter. It points at a
file the agent will not have, or its code compiles and never runs.

So hand it to a fresh agent that has no access to the source it was written
from, on a realistic project, and watch what it does.

aoh-skill-acceptance-test packages exactly that check. Run it before a skill
ships.
-->

---

# The workflow is a file

<div class="mt-8 w-[36rem]" style="--slidev-code-font-size: 1.5rem; --slidev-code-line-height: 2.4rem">

````md magic-move
```yaml
# openspec/config.yaml
schema: spec-driven
```
```yaml
# openspec/config.yaml
schema: aoh-spec-driven
```
````

</div>

<!--
THE WORKFLOW IS A FILE

Which steps the agent follows for every change is one line of configuration.

spec-driven is the default that ships with OpenSpec: proposal, specs, design,
tasks, then apply.

Click. Ours points at aoh-spec-driven, a schema that lives in this repository
under openspec/schemas.
-->

---

# Start from the default

<TermPanel>
<TermDots />
<PromptLine sigil="$">openspec schema fork spec-driven my-team-flow</PromptLine>
<div class="mt-3"><CheckLine>Forked 'spec-driven' to 'my-team-flow'</CheckLine></div>
<div class="font-mono text-sm mt-4 text-term-dim">openspec/schemas/my-team-flow/</div>
<div class="font-mono text-sm text-term-fg pl-4 leading-relaxed">
schema.yaml<br/>templates/proposal.md<br/>templates/spec.md<br/>templates/design.md<br/>templates/tasks.md
</div>
<PromptLine sigil="$" pad="mt-6">openspec schema validate my-team-flow</PromptLine>
<div class="mt-3"><CheckLine>Schema 'my-team-flow' is valid</CheckLine></div>
</TermPanel>

<!--
START FROM THE DEFAULT

This is where they start. fork copies the default schema into their own
repository: the schema file, and one template per document.

Then they edit the instructions to say what their team expects: their
platform, their conventions, what a reviewer always sends back.

validate checks the structure. Then config.yaml points at it.

Real output from OpenSpec 1.5, trimmed. It also prints that schema commands are
experimental, so say so if asked.
-->

---

```yaml
clicks: 1
```

# We added one step

<div class="flex items-start justify-center gap-16 mt-4">
<div class="flex flex-col items-center gap-3">
<div class="font-mono text-sm text-muted">spec-driven</div>
<FlowColumn :steps="['Proposal', 'Specs', 'Design', 'Tasks', 'Apply']" />
</div>
<div class="flex flex-col items-center gap-3">
<div class="font-mono text-sm text-muted">aoh-spec-driven</div>
<FlowColumn :steps="['Proposal', 'Specs', 'Design', 'Tasks', 'Lint', 'Apply']" :active="$clicks >= 1 ? 4 : -1" />
</div>
</div>

<!--
WE ADDED ONE STEP

Left, the default. Right, ours. The same four documents, then one more before
anything is built.

Click. Lint. It reads all four documents and checks them against the mistakes
agents make on this platform by default, forty-odd checks in ten groups:
roles attributed to Keycloak instead of AAS, a response that is not the AOH
envelope, an /api prefix on a path.

Every check is mechanical: pass, fail with the offending line quoted, or not
applicable. Apply will not start until it passes, so a mistake caught here
costs a document, not a build.
-->

---

# The gate, declared

<div class="mt-6 w-[42rem]" style="--slidev-code-font-size: 1.05rem; --slidev-code-line-height: 1.7rem">

```yaml {3-7|9-11}
artifacts:
  # proposal, specs, design and tasks, as before
  - id: lint
    generates: lint.md
    template: lint.md
    instruction: Check every document against known mistakes
    requires: [proposal, specs, design, tasks]

apply:
  requires: [lint]
  tracks: tasks.md
```

</div>

<!--
THE GATE, DECLARED

This is the whole mechanism. A step is an id, the file it produces, a template,
and an instruction the agent follows. requires puts them in order.

Click. apply.requires is the gate. The step that writes code cannot begin until
lint exists.

Abridged from openspec/schemas/aoh-spec-driven/schema.yaml. The real
instruction runs to a page.
-->

---

```yaml
clicks: 2
```

# Start small

<Points>
<NumStep :n="1">Put your stack in <span class="font-mono text-lg">config.yaml</span> as context and rules</NumStep>
<NumStep :n="2" class="transition-all duration-700" :class="$clicks >= 1 ? 'opacity-100' : 'opacity-0'">Fork a schema and rewrite its instructions</NumStep>
<NumStep :n="3" class="transition-all duration-700" :class="$clicks >= 2 ? 'opacity-100' : 'opacity-0'">Add a step that checks the others</NumStep>
</Points>

<!--
START SMALL

Nobody needs a schema on day one.

One. config.yaml takes a context block and rules per document, with no schema
at all. That is the fastest win: every proposal learns your stack.

Click. Two. When rules are not enough, fork the default and rewrite its
instructions.

Click. Three. When review keeps catching the same mistakes, turn them into a
step that checks for them, the way our lint does.

Skills follow the same path. Write the gotcha down first. Turn it into a skill
when a second project needs it.
-->
