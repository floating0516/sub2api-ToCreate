import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import quickstart from './quickstart'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  ...quickstart,
  admin,
  ...misc,
}
