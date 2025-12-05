import { useState, useCallback, useEffect, useRef } from 'react'
import { ProcessStep } from '../types'

interface ActiveProcess {
  type: string
  stepIndex: number
}

export function useProcessManager<T extends string>() {
  const [processes, setProcesses] = useState<Record<string, ActiveProcess>>({})
  const intervalsRef = useRef<Record<string, number>>({})

  useEffect(() => {
    return () => {
      Object.values(intervalsRef.current).forEach(clearInterval)
    }
  }, [])

  const runProcess = useCallback((
    entityId: string,
    type: T,
    stepsTemplate: ProcessStep[],
    onComplete?: () => void
  ) => {
    if (intervalsRef.current[entityId]) {
      clearInterval(intervalsRef.current[entityId])
      delete intervalsRef.current[entityId]
    }

    setProcesses(prev => ({ ...prev, [entityId]: { type, stepIndex: 0 } }))

    let currentStep = 0
    const totalSteps = stepsTemplate.length

    const interval = setInterval(() => {
      currentStep++
      
      setProcesses(prev => {
        if (!prev[entityId]) return prev
        return { ...prev, [entityId]: { type, stepIndex: currentStep } }
      })

      if (currentStep >= totalSteps - 1) {
        setTimeout(() => {
          clearInterval(interval)
          delete intervalsRef.current[entityId]
          onComplete?.()
          setProcesses(prev => {
            const next = { ...prev }
            delete next[entityId]
            return next
          })
        }, 1000)
      }
    }, 1500)

    intervalsRef.current[entityId] = interval
  }, [])

  const cancelProcess = useCallback((entityId: string) => {
    if (intervalsRef.current[entityId]) {
      clearInterval(intervalsRef.current[entityId])
      delete intervalsRef.current[entityId]
    }
    setProcesses(prev => {
      const next = { ...prev }
      delete next[entityId]
      return next
    })
  }, [])

  const getStepsWithState = useCallback((template: ProcessStep[], currentIndex: number): ProcessStep[] => {
    return template.map((step, index) => {
      let state: 'pending' | 'running' | 'done' = 'pending'
      if (index < currentIndex) state = 'done'
      else if (index === currentIndex) state = 'running'
      return { ...step, state }
    })
  }, [])

  return {
    processes,
    runProcess,
    cancelProcess,
    getStepsWithState
  }
}
