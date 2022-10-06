import Airplane from '../shared/Icons/Airplane';
import Event from '../shared/Icons/Event';
import Workflow from '../shared/Icons/Workflow';

// This defines the icons for each category, and the category sort order: once.
export const categoryMeta = {
  "getting started": {
    order: 1,
    icon: <Airplane fill="#fff" size={20} />
  },
  "sending & managing events": {
    order: 1,
    icon: <Event fill="#fff" size="20" />
  },
  "managing workflows": {
    order: 1,
    icon: <Workflow fill="#fff" size={20} />
  },
}
