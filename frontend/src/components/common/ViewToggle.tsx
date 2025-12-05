// import React from 'react'

interface ViewToggleProps {
  value: 'table' | 'card'
  onChange: (next: 'table' | 'card') => void
}

export default function ViewToggle({ value, onChange }: ViewToggleProps) {
  return (
    <div className="view-toggle-group">
      <button
        onClick={() => onChange('card')}
        className={`view-toggle-btn ${
          value === 'card' ? 'view-toggle-btn-active' : 'view-toggle-btn-inactive'
        }`}
      >
        Cards
      </button>
      <button
        onClick={() => onChange('table')}
        className={`view-toggle-btn ${
          value === 'table' ? 'view-toggle-btn-active' : 'view-toggle-btn-inactive'
        }`}
      >
        Table
      </button>
    </div>
  )
}
