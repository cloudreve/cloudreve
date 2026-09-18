import { Box } from "@mui/material";
import { useEffect, useRef } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { FBXLoader } from "three/examples/jsm/loaders/FBXLoader.js";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { OBJLoader } from "three/examples/jsm/loaders/OBJLoader.js";
import { fileExtension } from "../../../util";

export interface ModelViewerCanvasProps {
  src: string;
  fileName: string;
  onLoaded?: () => void;
  onError?: () => void;
}

const loadModel = (src: string, ext: string): Promise<THREE.Object3D> => {
  switch (ext) {
    case "gltf":
    case "glb":
      return new Promise((resolve, reject) =>
        new GLTFLoader().load(src, (g) => resolve(g.scene), undefined, reject),
      );
    case "fbx":
      return new Promise((resolve, reject) => new FBXLoader().load(src, resolve, undefined, reject));
    case "obj":
      return new Promise((resolve, reject) => new OBJLoader().load(src, resolve, undefined, reject));
    default:
      return Promise.reject(new Error(`unsupported model format: ${ext}`));
  }
};

const ModelViewerCanvas = ({ src, fileName, onLoaded, onError }: ModelViewerCanvasProps) => {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !src) {
      return;
    }

    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    container.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x1a1a1a);

    const camera = new THREE.PerspectiveCamera(50, 1, 0.01, 5000);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;

    scene.add(new THREE.HemisphereLight(0xffffff, 0x444444, 1.2));
    const dir = new THREE.DirectionalLight(0xffffff, 1.5);
    dir.position.set(3, 5, 4);
    scene.add(dir);
    scene.add(new THREE.GridHelper(10, 20, 0x444444, 0x2a2a2a));

    let disposed = false;
    let model: THREE.Object3D | undefined;
    loadModel(src, fileExtension(fileName) ?? "")
      .then((obj) => {
        if (disposed) {
          return;
        }
        model = obj;
        // Center on origin and scale so the model fits the view.
        const box = new THREE.Box3().setFromObject(obj);
        const center = box.getCenter(new THREE.Vector3());
        const size = box.getSize(new THREE.Vector3());
        const radius = Math.max(size.x, size.y, size.z, 1e-6);
        obj.position.sub(center);
        const scale = 4 / radius;
        obj.scale.setScalar(scale);
        scene.add(obj);

        const sphere = new THREE.Box3().setFromObject(obj).getBoundingSphere(new THREE.Sphere());
        camera.position.set(sphere.radius * 1.8, sphere.radius * 1.2, sphere.radius * 1.8);
        camera.near = sphere.radius / 100;
        camera.far = sphere.radius * 100;
        camera.updateProjectionMatrix();
        controls.target.set(0, 0, 0);
        controls.update();
        onLoaded?.();
      })
      .catch(() => onError?.());

    const resize = () => {
      const w = container.clientWidth;
      const h = container.clientHeight;
      if (w === 0 || h === 0) {
        return;
      }
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h);
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(container);

    const animate = () => {
      if (disposed) {
        return;
      }
      requestAnimationFrame(animate);
      controls.update();
      renderer.render(scene, camera);
    };
    animate();

    return () => {
      disposed = true;
      observer.disconnect();
      controls.dispose();
      model?.traverse((o) => {
        const mesh = o as THREE.Mesh;
        mesh.geometry?.dispose();
        const material = mesh.material as THREE.Material | THREE.Material[] | undefined;
        if (Array.isArray(material)) {
          material.forEach((m) => m.dispose());
        } else {
          material?.dispose();
        }
      });
      renderer.dispose();
      container.removeChild(renderer.domElement);
    };
  }, [src, fileName]);

  return <Box ref={containerRef} sx={{ width: "100%", height: "100%", minHeight: "calc(100vh - 200px)" }} />;
};

export default ModelViewerCanvas;
