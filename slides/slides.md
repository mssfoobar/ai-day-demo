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
<NumStep :n="3" pad="py-1">Components</NumStep>
<SubPoint>AGIL Ops Hub</SubPoint>
<SubPoint>Agent Skills</SubPoint>
<SubPoint>OpenSpec</SubPoint>
<NumStep :n="4" pad="py-1">Workshop</NumStep>
</Points>

<!--
CONTENTS

Thirty seconds. Read the five, do not expand on any of them.

The only one worth a beat is three: that is the part most of the room has
not seen before. Four is the demo, five is what they need to do before
the session.
-->

---

# Background

<Points>
<Bullet>Tasked to build a C2 application, as fast as AI could take us.</Bullet>
<Bullet>A C2 application needs an <span class="key">exorbitant amount of detail</span>.</Bullet>
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
<NumStep :n="1">Can we keep the results <span class="key">consistent</span> </NumStep>
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

# Components

<ArchDiagram class="mt-6" />

<!--
COMPONENTS

Start with the dashed box. Everything inside it runs on the developer's own
machine. Only the model sits outside, and that is the only thing that leaves.

The developer machine has the CLI installed. It hydrates the project with the
context and knowledge of the AGIL Ops Hub platform.

Click. That is what it installs. The knowledge and skills: code, design, code
review, test, devops, documentation. And OpenSpec, a lightweight framework
for spec-driven development.

Click. The coding harness reads those artifacts and feeds them to an LLM,
cloud or on-prem. That is what makes it consistently aware of the domain
knowledge before it writes anything.

The harness and the model are both swappable. The red box is the part that
is ours, and it carries over every time.
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
- that it is the middle layer from the Components slide, the only red box
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

This is the red box from the Components slide, on its own.

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

```yaml
clicks: 5
```

# How a change gets built {.text-center}

<div class="flex items-start justify-center gap-10 mt-4">
<div class="flex flex-col">
<div class="dgm-wide text-[10px] text-faint mb-3">One change, end to end</div>
<FlowColumn
  :steps="['Story', 'Propose spec', 'Review spec', 'Implement', 'Review build', 'Ship']"
  :active="$clicks"
/>
</div>

<!-- The artefacts only exist while the spec is being proposed and reviewed. -->
<div
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
</div>
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
because the arguing is over. The loop comes back to the next story.
-->

---

```yaml
clicks: 1
```

# Where you check the work {.text-center}

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

<div class="flex flex-col gap-2 mt-6 w-[44rem]">
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

# Setting up

<Points>
<NumStep :n="1">Install Node 24+, Go 1.25+ and Python 3</NumStep>
<NumStep :n="2">Install Podman, and start its machine</NumStep>
<NumStep :n="3" body="font-mono text-sm">npm install -g pnpm@10</NumStep>
<NumStep :n="4" body="font-mono text-sm">curl -fsSL https://claude.ai/install.sh | bash</NumStep>
<NumStep :n="5" body="font-mono text-sm">cd ai-day-demo && pnpm start</NumStep>
<NumStep :n="6">Open http://localhost:5173</NumStep>
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
