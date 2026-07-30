export const ORACLE_CHART_COLORS: string[] = [
  '#ff6b35',
  '#004e98',
  '#2d9b4e',
  '#ffd23f',
  '#e63946',
  '#50a4f5',
  '#5dbd72',
  '#ff8152',
  '#075fae',
  '#d9a900',
  '#f46170',
  '#706757'
]

export const ORACLE_CHART_NEUTRAL = '#9b9077'

export const ORACLE_CHART_SERIES = {
  orange: '#ff6b35',
  yellow: '#ffd23f',
  blue: '#004e98',
  lightBlue: '#50a4f5',
  green: '#2d9b4e',
  lightGreen: '#5dbd72',
  red: '#e63946',
  lightRed: '#f46170'
}

export function getOracleChartSurface(isDark: boolean) {
  return {
    text: isDark ? '#fffef5' : '#393137',
    mutedText: isDark ? '#c4b791' : '#706757',
    grid: isDark ? '#55485e' : '#eadfbd',
    tooltipBackground: isDark ? '#271f2f' : '#fffef5',
    tooltipTitle: isDark ? '#fffef5' : '#1a1423',
    tooltipBody: isDark ? '#ded3b5' : '#50483f'
  }
}
