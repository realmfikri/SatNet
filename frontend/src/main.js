import './style.css';
import * as Cesium from 'cesium';
import * as THREE from 'three';

const status = document.querySelector('#app');

status.innerHTML = `
  <main>
    <h1>SatNet Visual Sandbox</h1>
    <p>CesiumJS version: ${Cesium.VERSION}</p>
    <p>Three.js revision: ${THREE.REVISION}</p>
    <p class="note">Hook this page up to the backend simulation API to visualize the constellation.</p>
  </main>
`;
