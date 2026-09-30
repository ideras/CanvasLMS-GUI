<script>
  import StatusBar from './components/StatusBar.svelte'
  import CourseModal from './components/CourseModal.svelte'
  import CourseView from './components/CourseView.svelte'
  import UploadWizard from './components/UploadWizard.svelte'
  import SettingsModal from './components/SettingsModal.svelte'
  import { Toaster, toast } from 'svelte-sonner'
  import { Events } from '@wailsio/runtime'
  import { ListCourses, SelectCourse, GetConfig, GetCurrentCourse, NeedsSetup, ConfirmClose } from '../bindings/canvaslms-gui/app.js'

  let view = 'home'
  let currentCourse = null
  let currentAssignment = null
  let showCourseModal = false
  let showUploadWizard = false
  let showSettingsModal = false
  let settingsRequired = false   // true on first run (no cancel allowed)
  let courses = []
  let courseSet = []
  let uploadCourse = null
  let uploadAssignment = null
  let uploadSerial = 0
  let uploadActive = false
  let closeWarning = false
  let job = { state: '' }
  let uploadSummary = { total: 0, skipped: [], failed: [] }
  let jobStartedAt = 0
  let jobFinishedAt = 0
  let clock = Date.now()
  import { jobLabel, formatElapsed, summaryText, summaryDetails } from './lib/upload.js'
  $: jobElapsed = formatElapsed(jobStartedAt ? ((jobFinishedAt || clock) - jobStartedAt) / 1000 : 0)
  $: uploadStatus = job.state ? `Grades: ${jobLabel(job.state)}${jobStartedAt ? ' — ' + jobElapsed : ''}${!uploadActive ? ' — ' + summaryText(uploadSummary) : ''}` : ''

  async function loadCourses() {
    try {
      courses = await ListCourses()
    } catch (e) {
      toast.error(e.message || 'Failed to load courses')
    }
  }

  async function selectCourse(course) {
    try {
      currentCourse = await SelectCourse(course.id)
    } catch (e) {
      currentCourse = course
    }
    currentAssignment = null
    showCourseModal = false
    view = 'course'
  }

  async function openCourseModal() {
    await loadCourses()
    try {
      const cfg = await GetConfig()
      courseSet = cfg?.course_set || []
    } catch (e) { /* ignore */ }
    showCourseModal = true
  }

  function openUpload(assignment) {
    if (uploadActive) {
      toast.warning('A grade upload is already running. Finish it or stop it before starting another.')
      showUploadWizard = true
      return
    }
    currentAssignment = assignment
    uploadCourse = { ...currentCourse }
    uploadAssignment = { ...assignment }
    uploadSerial += 1
    showUploadWizard = true
  }

  async function initApp() {
    await loadCourses()
    try {
      const cfg = await GetConfig()
      courseSet = cfg?.course_set || []
    } catch (e) { /* ignore */ }
    try {
      const restored = await GetCurrentCourse()
      if (restored && restored.id) {
        currentCourse = restored
        view = 'course'
      }
    } catch (e) { /* ignore */ }
  }

  import { onMount } from 'svelte'
  onMount(() => {
    // Root listeners live for the application lifetime; store unsubscribe
    // functions so component teardown cannot clobber other listeners.
    const unsubAppError = Events.On('app:error', (e) => {
      toast.error(e.data.message)
    })
    const timer = setInterval(() => { clock = Date.now() }, 1000)
    const jobUnsubs = [
      Events.On('upload:started', () => {
        uploadActive = true
        job = { state: 'preparing' }
        jobStartedAt = 0
        jobFinishedAt = 0
        uploadSummary = { total: 0, skipped: [], failed: [] }
      }),
      Events.On('upload:batch_progress', (e) => {
        job = e.data
        jobStartedAt = Date.now() - (e.data.elapsed_seconds || 0) * 1000
        clock = Date.now()
      }),
      Events.On('upload:summary', (e) => { uploadSummary = e.data }),
      Events.On('upload:cancelled', () => { uploadActive = false; jobFinishedAt = Date.now(); job = { ...job, state: 'cancelled' } }),
      Events.On('upload:close_requested', () => { closeWarning = true }),
    ]
    const unsubUploadDone = Events.On('upload:done', (e) => {
      uploadActive = false
      uploadSummary = e.data
      jobFinishedAt = Date.now()
      job = { ...job, state: 'completed' }
      const options = { description: summaryDetails(e.data), duration: 12000 }
      if (e.data.failed?.length) toast.warning('Canvas completed: ' + summaryText(e.data), options)
      else toast.success('Canvas completed: ' + summaryText(e.data), options)
    })
    const unsubUploadError = Events.On('upload:error', (e) => {
      uploadActive = false
      jobFinishedAt = Date.now()
      job = { ...job, state: job.state === 'failed' ? 'failed' : job.state === 'submitting' ? 'unknown' : job.progress_url ? 'stopped' : 'failed' }
      toast.error(e.data.error || 'Upload failed', { description: summaryText(uploadSummary) + '\n' + summaryDetails(uploadSummary), duration: 12000 })
    })

    startApp()

    return () => {
      unsubAppError()
      unsubUploadDone()
      unsubUploadError()
      jobUnsubs.forEach(off => off())
      clearInterval(timer)
    }
  })

  async function startApp() {
    // Check whether a token is configured; if not, open settings first.
    try {
      const needsSetup = await NeedsSetup()
      if (needsSetup) {
        settingsRequired = true
        showSettingsModal = true
        return   // skip loading courses until setup is complete
      }
    } catch (_) { /* ignore */ }

    await initApp()
  }
</script>

<!-- Menu Bar -->
<header>
  <div class="title">CanvasLMS</div>
  <nav>
    <button class="{view === 'home' ? 'active' : ''}" on:click={() => view = 'home'}>
      Home
    </button>
    <button class="{view === 'course' ? 'active' : ''}" on:click={openCourseModal}>
      Courses
    </button>
  </nav>
  <div class="header-right">
    {#if currentCourse}
      <span class="course-badge">{currentCourse.name}</span>
    {/if}
    <button class="settings-btn" disabled={uploadActive} title={uploadActive ? 'Finish the grade upload before changing Canvas credentials' : 'Settings'} on:click={() => { settingsRequired = false; showSettingsModal = true }}>
      ⚙
    </button>
  </div>
</header>

<!-- Main Content -->
<main>
  {#if view === 'home'}
    <div class="home">
      <h1>Welcome to CanvasLMS</h1>
      <p>Select a course to get started.</p>
      <button class="primary" on:click={() => { openCourseModal() }}>
        Browse Courses
      </button>
    </div>

  {:else if view === 'course'}
    <CourseView
      course="{currentCourse}"
      onUpload={openUpload}
      onSwitchCourse={() => { openCourseModal() }}
    />
  {/if}
</main>

<StatusBar course="{currentCourse}" {uploadStatus} onUploadStatus={() => showUploadWizard = true}/>

{#if showCourseModal}
  <CourseModal
    courses="{courses}"
    {courseSet}
    onSelect={selectCourse}
    onClose={() => showCourseModal = false}
  />
{/if}

{#if uploadAssignment}
  {#key uploadSerial}
    <div hidden={!showUploadWizard}>
      <UploadWizard
        course={uploadCourse}
        assignment={uploadAssignment}
        onClose={() => showUploadWizard = false}
        onBackground={() => showUploadWizard = false}
      />
    </div>
  {/key}
{/if}

{#if closeWarning}
  <div class="modal-overlay">
    <div class="modal">
      <h2>Grade upload is still active</h2>
      <p>Closing stops local uploads and job monitoring. A grade job already submitted will continue in Canvas. Its final result will not be available here after exit.</p>
      <button class="secondary" on:click={() => closeWarning = false}>Keep app open</button>
      <button class="danger" on:click={() => { closeWarning = false; ConfirmClose() }}>Close app</button>
    </div>
  </div>
{/if}

{#if showSettingsModal}
  <SettingsModal
    required={settingsRequired}
    onClose={() => showSettingsModal = false}
    onSaved={async () => {
      showSettingsModal = false
      settingsRequired = false
      currentCourse = null
      view = 'home'
      await initApp()
    }}
  />
{/if}

<Toaster theme="dark" position="top-right" duration={4000} richColors closeButton />

<style>
  header {
    display: flex;
    align-items: center;
    gap: 24px;
    padding: 0 24px;
    height: 52px;
    background: linear-gradient(135deg, var(--frost-dark), #1a252f);
    color: white;
    flex-shrink: 0;
  }

  .title {
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 0.5px;
  }

  nav {
    display: flex;
    gap: 4px;
  }

  nav button {
    background: transparent;
    color: rgba(255,255,255,0.7);
    border-radius: 4px;
  }

  nav button:hover,
  nav button.active {
    background: rgba(255,255,255,0.12);
    color: white;
  }

  .header-right {
    margin-left: auto;
  }

  .course-badge {
    background: var(--frost-blue);
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .settings-btn {
    background: transparent;
    border: 1px solid rgba(255,255,255,0.25);
    color: rgba(255,255,255,0.8);
    border-radius: 6px;
    font-size: 16px;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    padding: 0;
    transition: background 0.15s, color 0.15s;
  }

  .settings-btn:hover {
    background: rgba(255,255,255,0.15);
    color: white;
  }

  main {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
  }

  .home {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    height: 100%;
  }

  .home h1 {
    font-size: 24px;
    color: var(--frost-dark);
  }

  .home p {
    color: var(--text-secondary);
  }
</style>
