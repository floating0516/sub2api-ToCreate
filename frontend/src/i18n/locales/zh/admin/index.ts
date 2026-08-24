import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import dailyReport from './dailyReport'
import audit from './audit'
import promptAudit from './promptAudit'
import plugins from './plugins'

export default {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...dailyReport,
  ...audit,
  ...promptAudit,
  ...plugins,
}
