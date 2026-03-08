import { useEffect, useState } from 'react'
import { ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'

const SCROLL_THRESHOLD = 300

export function ScrollToTop() {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const handleScroll = () => {
      setVisible(window.scrollY > SCROLL_THRESHOLD)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    handleScroll()
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  if (!visible) return null

  return (
    <Button
      variant="outline"
      size="icon"
      className="fixed bottom-6 right-6 z-50 size-10 rounded-full shadow-lg"
      onClick={scrollToTop}
      aria-label="Прокрутить вверх"
    >
      <ChevronUp className="size-5" />
    </Button>
  )
}
