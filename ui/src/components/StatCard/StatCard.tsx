import { Component, JSX } from 'solid-js';
import './StatCard.css';

interface StatCardProps {
  icon: JSX.Element;
  iconColor: string;
  value: string;
  label: string;
  subText?: string;
  subColor?: string;
}

export const StatCard: Component<StatCardProps> = (props) => {
  return (
    <div class="stat-card">
      <div class={`stat-card-icon ${props.iconColor}`}>
        {props.icon}
      </div>
      <div class="stat-card-info">
        <span class="stat-card-number">{props.value}</span>
        <span class="stat-card-label">{props.label}</span>
        {props.subText && (
          <span class={`stat-card-sub ${props.subColor || ''}`}>
            {props.subText}
          </span>
        )}
      </div>
    </div>
  );
};
