// Portraits for cards: each model is rendered once into a PNG by one shared offscreen renderer, then cached.
// Cards show plain images, so a page with many cards still uses a single, short-lived WebGL context.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import { Kit } from './kit'
import { troopModel } from './troops'
import { buildingModel } from './cityModels'

const cache = new Map<string, string>()
let renderer: THREE.WebGLRenderer | null = null
let kit: Kit | null = null
let idle = 0

function setup() {
  if (!renderer) {
    const c = document.createElement('canvas')
    renderer = new THREE.WebGLRenderer({ canvas: c, antialias: true, alpha: true, preserveDrawingBuffer: true })
    renderer.setPixelRatio(1)
    renderer.setSize(256, 256, false)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    renderer.setClearColor(0x000000, 0)
    kit = new Kit(true)
  }
  // Free the context a moment after the last request.
  clearTimeout(idle)
  idle = window.setTimeout(() => { kit?.dispose(); renderer?.dispose(); renderer?.forceContextLoss(); renderer = null; kit = null }, 1500)
  return { r: renderer, k: kit! }
}

function shoot(model: THREE.Object3D, zoom = 1): string {
  const { r, k } = setup()
  const scene = new THREE.Scene()
  k.lights(scene, 1.05)
  scene.add(model)
  const box = new THREE.Box3().setFromObject(model)
  const size = box.getSize(new THREE.Vector3()), c = box.getCenter(new THREE.Vector3())
  const span = Math.max(size.x, size.y, size.z) * 0.62 * zoom
  const cam = new THREE.OrthographicCamera(-span, span, span, -span, 0.1, 100)
  const dir = new THREE.Vector3(Math.sin(Math.PI / 4) * Math.cos(0.42), Math.sin(0.42), Math.cos(Math.PI / 4) * Math.cos(0.42))
  cam.position.copy(c).addScaledVector(dir, 30)
  cam.lookAt(c)
  r.render(scene, cam)
  return r.domElement.toDataURL('image/png')
}

/** troopPortrait: a data URL of the troop, or null when WebGL is unavailable. */
export function troopPortrait(type: string): string | null {
  const key = 't:' + type
  if (cache.has(key)) return cache.get(key)!
  try {
    const { k } = setup()
    const url = shoot(troopModel(k, type), 0.72)
    cache.set(key, url)
    return url
  } catch { return null }
}

/** buildingPortrait: a building of a given level for the build list. */
export function buildingPortrait(type: BuildingType, level = 1): string | null {
  const key = `b:${type}:${level}`
  if (cache.has(key)) return cache.get(key)!
  try {
    const { k } = setup()
    const url = shoot(buildingModel(k, type, level, { e: type === 'wall', s: false }, { smoke: [] }), 1)
    cache.set(key, url)
    return url
  } catch { return null }
}
