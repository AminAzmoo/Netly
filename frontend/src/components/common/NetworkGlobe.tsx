import { useEffect, useRef, useMemo, useState } from 'react'
import Globe from 'react-globe.gl'

interface Node {
  id: string
  name: string
  lat: number
  lng: number
  status: string
  role: string
}

interface Link {
  source: string
  target: string
  status: string
}

interface NetworkGlobeProps {
  nodes: Node[]
  links: Link[]
}

// Helper to get CSS variable value
const getCssVar = (name: string) => {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

export default function NetworkGlobe({ nodes, links }: NetworkGlobeProps) {
  const globeRef = useRef<any>(null)
  const [countries, setCountries] = useState<any>({ features: [] })
  
  // Theme colors state to hold values from CSS variables
  const [themeColors, setThemeColors] = useState({
    bgBase: '',
    neonA: '',
    neonB: '',
    oceanColor: '',
    landColor: '',
    landColor2: '',
    landGlow: '',
    error: '',
    warning: '',
    offline: '',
  })

  // Initialize theme colors from CSS variables
  useEffect(() => {
    const colors = {
      bgBase: getCssVar('--color-bg-base'),
      neonA: getCssVar('--color-neon-a'),
      neonB: getCssVar('--color-neon-b'),
      oceanColor: getCssVar('--color-globe-ocean'),
      landColor: getCssVar('--color-land-1'),
      landColor2: getCssVar('--color-land-2'),
      landGlow: getCssVar('--color-neon-a'),
      error: getCssVar('--color-error'),
      warning: getCssVar('--color-warning'),
      offline: getCssVar('--color-node-offline'),
    }
    setThemeColors(colors)
  }, [])


  // Load countries GeoJSON
  useEffect(() => {
    fetch('https://raw.githubusercontent.com/vasturiano/react-globe.gl/master/example/datasets/ne_110m_admin_0_countries.geojson')
      .then((res) => res.json())
      .then(setCountries)
  }, [])

  // Convert nodes to points data
  const pointsData = useMemo(
    () =>
      nodes.map((node) => ({
        lat: node.lat,
        lng: node.lng,
        name: node.name,
        status: node.status,
        size: node.role === 'Core' ? 0.8 : 0.5,
        color:
          node.status === 'Online'
            ? themeColors.neonA
            : node.status === 'Degraded'
              ? themeColors.neonB
              : themeColors.offline,
      })),
    [nodes, themeColors]
  )

  // Convert links to arcs data
  const arcsData = useMemo(
    () =>
      links.map((link) => {
        const sourceNode = nodes.find((n) => n.name === link.source)
        const targetNode = nodes.find((n) => n.name === link.target)
        return {
          startLat: sourceNode?.lat || 0,
          startLng: sourceNode?.lng || 0,
          endLat: targetNode?.lat || 0,
          endLng: targetNode?.lng || 0,
          color:
            link.status === 'Live'
              ? themeColors.neonA
              : link.status === 'Configuring'
                ? themeColors.warning
                : themeColors.error,
          status: link.status,
        }
      }),
    [links, nodes, themeColors]
  )

  useEffect(() => {
    if (globeRef.current) {
      globeRef.current.controls().autoRotate = true
      globeRef.current.controls().autoRotateSpeed = 0.5
    }
  }, [])

  return (
    <div className="globe-container-full">
      <Globe
        ref={globeRef}
        backgroundColor="rgba(0,0,0,0)"
        showGlobe={true}
        showAtmosphere={true}
        atmosphereColor={themeColors.neonA}
        atmosphereAltitude={0.2}
        globeImageUrl=""
        hexPolygonsData={countries.features}
        hexPolygonResolution={3}
        hexPolygonMargin={0.4}
        hexPolygonUseDots={false}
        hexPolygonColor={(d: any) => {
          const val = (d.properties?.ISO_A3?.charCodeAt(0) || 0) + (d.properties?.ISO_A3?.charCodeAt(1) || 0);
          return val % 2 === 0 ? themeColors.landColor : themeColors.landColor2;
        }}
        hexPolygonAltitude={0.008}
        hexPolygonCurvatureResolution={5}
        onGlobeReady={() => {
          if (globeRef.current) {
            // Safely handle globeMaterial access
            let globeMaterial;
            try {
              if (typeof globeRef.current.globeMaterial === 'function') {
                globeMaterial = globeRef.current.globeMaterial();
              } else {
                globeMaterial = globeRef.current.globeMaterial;
              }

              if (globeMaterial) {
                if (themeColors.oceanColor) globeMaterial.color.set(themeColors.oceanColor);
                if (themeColors.landGlow) {
                  globeMaterial.emissive.set(themeColors.landGlow);
                  globeMaterial.emissiveIntensity = 0.06;
                }
              }
            } catch (e) {
              console.warn('Failed to apply globe material:', e);
            }
          }
        }}
        pointsData={pointsData}
        pointLat="lat"
        pointLng="lng"
        pointColor="color"
        pointAltitude={0.009}
        pointRadius="size"
        pointLabel="name"
        arcsData={arcsData}
        arcColor="color"
        arcAltitude={0}
        arcDashLength={0.4}
        arcDashGap={0.2}
        arcDashAnimateTime={1500}
        arcStroke={0.5}
      />
    </div>
  )
}
