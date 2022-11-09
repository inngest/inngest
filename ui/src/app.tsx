import './index.css'
import Header from './components/Header'
import Sidebar from './components/Sidebar'
import SidebarLink from './components/SidebarLink'
import Content from './components/Content'
import { IconFeed, IconBook } from './icons'

export function App() {
  return (
    <div class="w-screen h-screen text-slate-400 text-sm grid grid-cols-app grid-rows-app overflow-hidden">
      <Header />
      <Sidebar>
        <SidebarLink icon={<IconFeed />} active badge={20} />
        <SidebarLink icon={<IconBook />} />
      </Sidebar>
      <Content />
    </div>
  )
}
