import './style.css';
import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';

const EARTH_RADIUS_KM = 6371;
const KM_TO_UNITS = 1; // 1 km == 1 unit for clarity

const app = document.querySelector('#app');
app.innerHTML = `
  <div class="page">
    <header class="hero">
      <div>
        <p class="eyebrow">SatNet Frontend</p>
        <h1>Orbital Operations Dashboard</h1>
        <p class="lede">Live Earth view with satellite orbits, coverage heatmap, and real-time controls.</p>
      </div>
      <div class="status" id="connection-status">Connecting to data feeds…</div>
    </header>

    <section class="layout">
      <div class="controls">
        <div class="control-card">
          <div class="card-header">
            <h2>Orbit Designer</h2>
            <p>Create or adjust a target orbit and push it to the backend.</p>
          </div>
          <label class="slider">
            <div>
              <span>Altitude</span>
              <span id="altitude-value">550 km</span>
            </div>
            <input id="altitude" type="range" min="300" max="1200" step="10" value="550" />
          </label>
          <label class="slider">
            <div>
              <span>Inclination</span>
              <span id="inclination-value">53°</span>
            </div>
            <input id="inclination" type="range" min="0" max="98" step="1" value="53" />
          </label>
          <button id="push-orbit" class="primary">Send orbit to backend</button>
        </div>

        <div class="control-card">
          <div class="card-header">
            <h2>Constellation actions</h2>
            <p>Test fault handling and refresh coverage.</p>
          </div>
          <div class="button-row">
            <button id="refresh-coverage">Refresh coverage map</button>
            <button id="destroy-sat" class="danger">Destroy random satellite</button>
          </div>
          <div class="feed-status">
            <div>
              <span class="dot live" id="ws-dot"></span>
              <span>Telemetry feed</span>
            </div>
            <div>
              <span class="dot" id="rest-dot"></span>
              <span>REST backend</span>
            </div>
          </div>
          <p class="hint">Data endpoints: <code>/api/satellites</code>, <code>/api/coverage</code>, <code>/api/orbits</code>, <code>/ws/telemetry</code>.</p>
        </div>
      </div>

      <div class="viewer-card">
        <div class="viewer" id="viewer"></div>
        <div class="legend">
          <div><span class="chip orbit"></span> Orbit path</div>
          <div><span class="chip satellite"></span> Satellite</div>
          <div><span class="chip coverage"></span> Coverage heatmap</div>
        </div>
      </div>
    </section>
  </div>
`;

const connectionStatus = document.querySelector('#connection-status');
const restDot = document.querySelector('#rest-dot');
const wsDot = document.querySelector('#ws-dot');
const altitudeInput = document.querySelector('#altitude');
const inclinationInput = document.querySelector('#inclination');
const altitudeValue = document.querySelector('#altitude-value');
const inclinationValue = document.querySelector('#inclination-value');

let renderer;
let camera;
let scene;
let coverageMesh;
let telemetrySocket;

const orbitPaths = new Map();
const satellites = new Map();

function updateLabel(input, display, suffix) {
  display.textContent = `${Number(input.value).toLocaleString()}${suffix}`;
}

updateLabel(altitudeInput, altitudeValue, ' km');
updateLabel(inclinationInput, inclinationValue, '°');

function setStatus(message, isError = false) {
  connectionStatus.textContent = message;
  connectionStatus.classList.toggle('error', isError);
}

function toggleDot(dot, active) {
  dot.classList.toggle('live', active);
}

function latLonAltToCartesian(lat, lon, altKm) {
  const radius = (EARTH_RADIUS_KM + altKm) * KM_TO_UNITS;
  const phi = THREE.MathUtils.degToRad(90 - lat);
  const theta = THREE.MathUtils.degToRad(lon + 180);
  const x = radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.cos(phi);
  const z = radius * Math.sin(phi) * Math.sin(theta);
  return new THREE.Vector3(x, y, z);
}

function buildScene() {
  const container = document.querySelector('#viewer');
  renderer = new THREE.WebGLRenderer({ antialias: true });
  renderer.setPixelRatio(window.devicePixelRatio);
  renderer.setSize(container.clientWidth, container.clientHeight);
  container.appendChild(renderer.domElement);

  scene = new THREE.Scene();
  scene.background = new THREE.Color('#020617');

  camera = new THREE.PerspectiveCamera(45, container.clientWidth / container.clientHeight, 0.1, 20000);
  camera.position.set(0, -12000, 6000);

  const controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.target.set(0, 0, 0);

  const ambient = new THREE.AmbientLight(0xffffff, 0.45);
  scene.add(ambient);
  const directional = new THREE.DirectionalLight(0xffffff, 0.7);
  directional.position.set(5000, 5000, 5000);
  scene.add(directional);

  const earthGeometry = new THREE.SphereGeometry(EARTH_RADIUS_KM, 64, 64);
  const earthMaterial = new THREE.MeshPhongMaterial({
    color: new THREE.Color('#0ea5e9'),
    emissive: new THREE.Color('#0b1224'),
    shininess: 10,
    specular: new THREE.Color('#38bdf8'),
  });
  const earth = new THREE.Mesh(earthGeometry, earthMaterial);
  scene.add(earth);

  const atmosphereGeometry = new THREE.SphereGeometry(EARTH_RADIUS_KM + 50, 64, 64);
  const atmosphereMaterial = new THREE.MeshBasicMaterial({
    color: new THREE.Color('#67e8f9'),
    transparent: true,
    opacity: 0.08,
    blending: THREE.AdditiveBlending,
  });
  const atmosphere = new THREE.Mesh(atmosphereGeometry, atmosphereMaterial);
  scene.add(atmosphere);

  coverageMesh = buildCoverageMesh();
  scene.add(coverageMesh);

  window.addEventListener('resize', () => {
    const { clientWidth, clientHeight } = container;
    camera.aspect = clientWidth / clientHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(clientWidth, clientHeight);
  });

  function animate() {
    requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
  }
  animate();
}

function buildCoverageMesh() {
  const coverageGeometry = new THREE.SphereGeometry(EARTH_RADIUS_KM + 80, 90, 45);
  const colors = [];
  const color = new THREE.Color();

  for (let i = 0; i < coverageGeometry.attributes.position.count; i += 1) {
    color.setRGB(0.1, 0.3, 0.6);
    colors.push(color.r, color.g, color.b);
  }
  coverageGeometry.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));

  return new THREE.Mesh(
    coverageGeometry,
    new THREE.MeshPhongMaterial({
      vertexColors: true,
      transparent: true,
      opacity: 0.45,
      emissive: '#0ea5e9',
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    }),
  );
}

function updateCoverageHeatmap(samples = []) {
  if (!coverageMesh) return;
  const { geometry } = coverageMesh;
  const colors = geometry.attributes.color;
  const color = new THREE.Color();
  const defaultFade = 0.18;

  for (let i = 0; i < colors.count; i += 1) {
    const vertex = new THREE.Vector3().fromBufferAttribute(geometry.attributes.position, i).normalize();
    const lat = 90 - THREE.MathUtils.radToDeg(Math.acos(vertex.y));
    const lon = THREE.MathUtils.radToDeg(Math.atan2(vertex.z, vertex.x)) - 180;

    let strength = defaultFade;
    for (const sample of samples) {
      const distance = Math.hypot(sample.lat - lat, sample.lon - lon);
      if (distance < sample.radiusDeg) {
        const proximity = 1 - distance / sample.radiusDeg;
        strength = Math.max(strength, sample.intensity * proximity);
      }
    }

    color.setHSL(0.5 * (1 - strength), 0.8, Math.max(0.3, strength));
    colors.setXYZ(i, color.r, color.g, color.b);
  }
  colors.needsUpdate = true;
}

function ensureOrbitPath(id, altitude, inclination) {
  const radius = (EARTH_RADIUS_KM + altitude) * KM_TO_UNITS;
  const points = [];
  for (let angle = 0; angle <= 360; angle += 3) {
    const rad = THREE.MathUtils.degToRad(angle);
    const x = radius * Math.cos(rad);
    const y = 0;
    const z = radius * Math.sin(rad);
    const point = new THREE.Vector3(x, y, z);
    point.applyAxisAngle(new THREE.Vector3(1, 0, 0), THREE.MathUtils.degToRad(inclination));
    points.push(point);
  }
  const geometry = new THREE.BufferGeometry().setFromPoints(points);
  const material = new THREE.LineDashedMaterial({ color: '#22d3ee', dashSize: 500, gapSize: 300 });
  const line = new THREE.Line(geometry, material);
  line.computeLineDistances();

  const existing = orbitPaths.get(id);
  if (existing) scene.remove(existing);
  orbitPaths.set(id, line);
  scene.add(line);
}

function ensureSatellite(id, color = '#f8fafc') {
  if (satellites.has(id)) return satellites.get(id);
  const geometry = new THREE.SphereGeometry(50, 16, 16);
  const material = new THREE.MeshStandardMaterial({ color, emissive: '#f97316', roughness: 0.4 });
  const mesh = new THREE.Mesh(geometry, material);
  satellites.set(id, mesh);
  scene.add(mesh);
  return mesh;
}

function updateSatellitePosition(sat, timestampMs) {
  const { inclination = 0, altitude = 550, meanMotion = 0.05, raan = 0 } = sat;
  const period = (2 * Math.PI) / meanMotion;
  const time = (timestampMs / 1000) % period;
  const trueAnomaly = time * meanMotion;
  const radius = (EARTH_RADIUS_KM + altitude) * KM_TO_UNITS;

  const x = radius * Math.cos(trueAnomaly);
  const y = 0;
  const z = radius * Math.sin(trueAnomaly);
  const position = new THREE.Vector3(x, y, z);
  position.applyAxisAngle(new THREE.Vector3(1, 0, 0), THREE.MathUtils.degToRad(inclination));
  position.applyAxisAngle(new THREE.Vector3(0, 1, 0), THREE.MathUtils.degToRad(raan));

  const mesh = ensureSatellite(sat.id, sat.color);
  mesh.position.copy(position);
  ensureOrbitPath(sat.id, altitude, inclination);
}

function setConstellation(data = []) {
  const now = Date.now();
  data.forEach((sat, index) => {
    const enrichedSat = { ...sat, color: sat.color || `hsl(${(index * 57) % 360}, 90%, 70%)` };
    updateSatellitePosition(enrichedSat, now);
  });
}

async function fetchJSON(url, options = {}) {
  try {
    const response = await fetch(url, options);
    if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
    toggleDot(restDot, true);
    return await response.json();
  } catch (error) {
    console.error('REST error', error);
    toggleDot(restDot, false);
    setStatus('Backend unreachable — running in demo mode.', true);
    return null;
  }
}

async function fetchConstellation() {
  const payload = await fetchJSON('/api/satellites');
  if (payload && Array.isArray(payload.satellites)) {
    setStatus('Constellation synced via REST.');
    setConstellation(payload.satellites);
  } else {
    const demo = Array.from({ length: 12 }).map((_, i) => ({
      id: `demo-${i + 1}`,
      altitude: 550 + (i % 3) * 50,
      inclination: 53 + (i % 2) * 10,
      meanMotion: 0.05 + i * 0.0015,
      raan: (i * 15) % 180,
    }));
    setConstellation(demo);
  }
}

async function fetchCoverage() {
  const payload = await fetchJSON('/api/coverage');
  if (payload && Array.isArray(payload.cells)) {
    updateCoverageHeatmap(payload.cells);
    setStatus('Coverage heatmap refreshed.');
    return;
  }

  // demo coverage wave
  const samples = [];
  for (let i = 0; i < 12; i += 1) {
    samples.push({
      lat: Math.sin(Date.now() / 20000 + i) * 60,
      lon: (i * 30 + (Date.now() / 1200)) % 360 - 180,
      radiusDeg: 22,
      intensity: 0.7,
    });
  }
  updateCoverageHeatmap(samples);
}

async function pushOrbit() {
  const altitude = Number(altitudeInput.value);
  const inclination = Number(inclinationInput.value);

  const body = JSON.stringify({ altitude, inclination });
  const payload = await fetchJSON('/api/orbits', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  });

  if (payload) {
    setStatus(`Orbit pushed: ${altitude} km @ ${inclination}°.`);
    setConstellation(payload.satellites || []);
  }
}

async function destroyRandomSatellite() {
  const payload = await fetchJSON('/api/satellites/destroy', { method: 'POST' });
  if (payload) {
    setStatus(payload.message || 'Random satellite destroyed.');
    setConstellation(payload.satellites || []);
  }
}

function connectTelemetry() {
  if (telemetrySocket) telemetrySocket.close();
  telemetrySocket = new WebSocket(`ws://${window.location.host}/ws/telemetry`);

  telemetrySocket.addEventListener('open', () => {
    setStatus('Telemetry stream connected.');
    toggleDot(wsDot, true);
  });

  telemetrySocket.addEventListener('message', (event) => {
    try {
      const payload = JSON.parse(event.data);
      if (Array.isArray(payload.satellites)) {
        setConstellation(payload.satellites);
      }
      if (Array.isArray(payload.coverage)) {
        updateCoverageHeatmap(payload.coverage);
      }
    } catch (error) {
      console.error('WS parse error', error);
    }
  });

  telemetrySocket.addEventListener('close', () => {
    toggleDot(wsDot, false);
    setStatus('Telemetry disconnected — retrying…', true);
    setTimeout(connectTelemetry, 3000);
  });

  telemetrySocket.addEventListener('error', () => {
    toggleDot(wsDot, false);
    setStatus('Telemetry error — running in demo mode.', true);
  });
}

function setupEventHandlers() {
  altitudeInput.addEventListener('input', () => updateLabel(altitudeInput, altitudeValue, ' km'));
  inclinationInput.addEventListener('input', () => updateLabel(inclinationInput, inclinationValue, '°'));
  document.querySelector('#push-orbit').addEventListener('click', pushOrbit);
  document.querySelector('#refresh-coverage').addEventListener('click', fetchCoverage);
  document.querySelector('#destroy-sat').addEventListener('click', destroyRandomSatellite);
}

function boot() {
  buildScene();
  setupEventHandlers();
  fetchConstellation();
  fetchCoverage();
  connectTelemetry();
}

boot();
