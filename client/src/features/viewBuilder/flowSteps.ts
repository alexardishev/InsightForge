import type { FlowStep } from '../../components/FlowLayout';

export const builderSteps: FlowStep[] = [
  { title: 'Source & Schema', description: 'Подключение и выбор схемы' },
  { title: 'Tables & Columns', description: 'Отбор сущностей' },
  { title: 'Joins', description: 'Связи без cartesian join' },
  { title: 'Transforms', description: 'Переименования и обогащение' },
  { title: 'Review', description: 'Проверка и запуск' },
];
